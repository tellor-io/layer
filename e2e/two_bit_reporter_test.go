package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestTwoBitAgentReportCreditsReporter asks an agent to report for a reporter.
// The agent cannot submit until it accepts. After it accepts, a 2bitAgentic
// report and its tip belong to the reporter. Any other query type still fails.
func TestTwoBitAgentReportCreditsReporter(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	reporterNode := validators[0].Node
	reporterAddr := validators[0].AccAddr

	agent := interchaintest.GetAndFundTestUsers(t, ctx, "two-bit-agent", math.NewInt(10_000*1e6), chain)[0]
	agentAddr := agent.FormattedAddress()

	txHash, err := reporterNode.ExecTx(ctx, reporterAddr, "reporter", "create-reporter",
		"0.1", "1000000", "two-bit-owner",
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	fmt.Println("TX HASH (create reporter): ", txHash)

	txHash, err = reporterNode.ExecTx(ctx, reporterAddr, "reporter", "register-two-bit-reporter", agentAddr,
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "400000", "--fees", "50loya")
	require.NoError(err)
	fmt.Println("TX HASH (register 2bit agent): ", txHash)

	spec := e2e.DataSpec{
		DocumentHash:      "2bit-agentic",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		AbiComponents: []*registrytypes.ABIComponent{{
			Name:            "payload",
			FieldType:       "string",
			NestedComponent: []*registrytypes.ABIComponent{},
		}},
		Registrar:         reporterAddr,
		QueryType:         "2bitAgentic",
		ReportBlockWindow: "20",
	}
	specBz, err := json.Marshal(spec)
	require.NoError(err)
	txHash, err = reporterNode.ExecTx(ctx, reporterAddr, "registry", "register-spec", "2bitAgentic", string(specBz),
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	fmt.Println("TX HASH (register 2bitAgentic spec): ", txHash)

	queryBz, _, err := e2e.QueryWithTimeout(ctx, reporterNode, "registry", "generate-querydata", "2bitAgentic", `["report"]`)
	require.NoError(err)
	var queryData e2e.QueryGenerateQuerydataResponse
	require.NoError(json.Unmarshal(queryBz, &queryData))

	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	txHash, err = reporterNode.ExecTx(ctx, reporterAddr, "oracle", "tip", queryData.QueryData, tip.String(),
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	fmt.Println("TX HASH (tip 2bitAgentic query): ", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 1, reporterNode))

	value := layerutil.EncodeValue(123.45)
	// A request is not a registration. The agent cannot report until it accepts.
	_, err = reporterNode.ExecTx(ctx, agentAddr, "oracle", "submit-value", queryData.QueryData, value,
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "1000000", "--fees", "50loya")
	require.Error(err)

	txHash, err = reporterNode.ExecTx(ctx, agentAddr, "reporter", "accept-two-bit-registration", reporterAddr,
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "400000", "--fees", "50loya")
	require.NoError(err)
	fmt.Println("TX HASH (accept 2bit registration): ", txHash)

	spotBz, _, err := e2e.QueryWithTimeout(ctx, reporterNode, "registry", "generate-querydata", "spotprice", `["eth","usd"]`)
	require.NoError(err)
	var spotData e2e.QueryGenerateQuerydataResponse
	require.NoError(json.Unmarshal(spotBz, &spotData))
	_, err = reporterNode.ExecTx(ctx, agentAddr, "oracle", "submit-value", spotData.QueryData, value,
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "1000000", "--fees", "50loya")
	require.Error(err)

	txHash, err = reporterNode.ExecTx(ctx, agentAddr, "oracle", "submit-value", queryData.QueryData, value,
		"--keyring-dir", reporterNode.HomeDir(), "--gas", "1000000", "--fees", "50loya")
	require.NoError(err)
	fmt.Println("TX HASH (agent submits value): ", txHash)

	reportsBz, _, err := e2e.QueryWithTimeout(ctx, reporterNode, "oracle", "get-reportsby-reporter", reporterAddr, "--page-limit", "1")
	require.NoError(err)
	var reporterReports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(reportsBz, &reporterReports))
	require.Len(reporterReports.MicroReports, 1)
	report := reporterReports.MicroReports[0]
	require.Equal(reporterAddr, report.Reporter)
	require.Equal("2bitAgentic", report.QueryType)
	require.Equal(value, report.Value)
	require.NotEmpty(report.Power)

	agentReportsBz, _, err := e2e.QueryWithTimeout(ctx, reporterNode, "oracle", "get-reportsby-reporter", agentAddr, "--page-limit", "1")
	require.NoError(err)
	var agentReports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(agentReportsBz, &agentReports))
	require.Empty(agentReports.MicroReports)

	byQueryBz, _, err := e2e.QueryWithTimeout(ctx, reporterNode, "oracle", "get-reportsby-qid", report.QueryID, "--page-limit", "5")
	require.NoError(err)
	var queryReports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(byQueryBz, &queryReports))
	require.Len(queryReports.MicroReports, 1)
	require.Equal(reporterAddr, queryReports.MicroReports[0].Reporter)

	// The query expires 20 blocks after the tip. Waiting past that aggregates it
	// and pays the tip to the address stored on the report.
	require.NoError(testutil.WaitForBlocks(ctx, 22, reporterNode))

	// Two percent of the tip is burned. The rest is paid to the reporter.
	expectedTip := tipAmt.MulRaw(98).QuoRaw(100)
	reporterTips := queryAvailableTips(t, ctx, reporterNode, reporterAddr)
	require.True(reporterTips.Equal(math.LegacyNewDecFromInt(expectedTip)), "reporter tips %s, expected %s", reporterTips, expectedTip)

	agentTips := queryAvailableTips(t, ctx, reporterNode, agentAddr)
	require.True(agentTips.IsZero(), "agent tips %s", agentTips)
}

func queryAvailableTips(t *testing.T, ctx context.Context, node *cosmos.ChainNode, selector string) math.LegacyDec {
	t.Helper()
	bz, _, err := e2e.QueryWithTimeout(ctx, node, "reporter", "available-tips", selector)
	require.NoError(t, err)
	var resp struct {
		AvailableTips string `json:"available_tips"`
	}
	require.NoError(t, json.Unmarshal(bz, &resp))
	if resp.AvailableTips == "" {
		return math.LegacyZeroDec()
	}
	tips, err := math.LegacyNewDecFromStr(resp.AvailableTips)
	require.NoError(t, err)
	return tips
}
