package e2e_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"
)

const (
	qData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003626368000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value = "000000000000000000000000000000000000000000000058528649cf80ee0000"
)

type MicroReport struct {
	Reporter        string `json:"reporter"`
	Power           string `json:"power"`
	QueryType       string `json:"query_type"`
	QueryID         string `json:"query_id"`
	AggregateMethod string `json:"aggregate_method"`
	Value           string `json:"value"`
	Timestamp       string `json:"timestamp"`
	BlockNumber     string `json:"block_number"`
}

type ReportsResponse struct {
	MicroReports []MicroReport `json:"microReports"`
}
type AggregateReport struct {
	Aggregate struct {
		QueryID           string `json:"query_id"`
		AggregateValue    string `json:"aggregate_value"`
		AggregateReporter string `json:"aggregate_reporter"`
		ReporterPower     string `json:"reporter_power"`
		Reporters         []struct {
			Reporter    string `json:"reporter"`
			Power       string `json:"power"`
			BlockNumber string `json:"block_number"`
		} `json:"reporters"`
		Index       string `json:"index"`
		Height      string `json:"height"`
		MicroHeight string `json:"micro_height"`
		MetaID      string `json:"meta_id"`
	} `json:"aggregate"`
	Timestamp string `json:"timestamp"`
}

type Proposal struct {
	Messages  []map[string]interface{} `json:"messages"`
	Metadata  string                   `json:"metadata"`
	Deposit   string                   `json:"deposit"`
	Title     string                   `json:"title"`
	Summary   string                   `json:"summary"`
	Expedited bool                     `json:"expedited"`
}

func ExecProposal(ctx context.Context, keyName string, prop Proposal, tn *cosmos.ChainNode) (string, error) {
	content, err := json.Marshal(prop)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	proposalFilename := fmt.Sprintf("%x.json", hash)
	err = tn.WriteFile(ctx, content, proposalFilename)
	if err != nil {
		return "", fmt.Errorf("writing param change proposal: %w", err)
	}

	proposalPath := filepath.Join(tn.HomeDir(), proposalFilename)

	command := []string{
		"gov", "submit-proposal",
		proposalPath,
	}

	return tn.ExecTx(ctx, keyName, command...)
}

func TestLearn(t *testing.T) {
	ctx := context.Background()
	layer := e2e.LayerSpinup(t) // *cosmos.CosmosChain type
	validatorI := layer.Validators[0]

	valAddress, err := validatorI.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	// sample of how add a user and fund it
	user := interchaintest.GetAndFundTestUsers(t, ctx, "user1", math.OneInt(), layer)[0]
	fmt.Println("User address: ", user.FormattedAddress())

	prop := Proposal{
		Messages: []map[string]interface{}{
			{
				"@type":     "/layer.mint.MsgInit",
				"authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx",
			},
		},
		Metadata:  "ipfs://CID",
		Deposit:   "50000000loya",
		Title:     "Init tbr minting",
		Summary:   "Initialize inflationary rewards",
		Expedited: false,
	}
	_, err = ExecProposal(ctx, "validator", prop, validatorI)
	require.NoError(t, err)

	for _, v := range layer.Validators {
		_, err = v.ExecTx(ctx, "validator", "gov", "vote", "1", "yes", "--gas", "1000000", "--fees", "1000000loya", "--keyring-dir", layer.HomeDir())
		require.NoError(t, err)
	}
	err = testutil.WaitForBlocks(ctx, 3, validatorI)
	require.NoError(t, err)

	result, err := layer.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	fmt.Println("Proposal result: ", result)
	// create reporter
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", valAddress, qData, "1000000loya", "--keyring-dir", layer.HomeDir()), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)

	txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valAddress, qData, value, "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	res1, _, err := validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAddress)
	require.NoError(t, err)

	var microReports ReportsResponse
	err = json.Unmarshal(res1, &microReports)
	require.NoError(t, err)
	fmt.Println("Micro reports: ", microReports)
	require.Equal(t, microReports.MicroReports[0].Reporter, valAddress)

	// get aggreate report
	qidbz, err := utils.QueryIDFromDataString(qData)
	require.NoError(t, err)

	res2, _, err := validatorI.ExecQuery(ctx, "oracle", "get-current-aggregate-report", hex.EncodeToString(qidbz))
	require.NoError(t, err)

	var aggReport AggregateReport
	fmt.Println("Aggregate report: ", string(res2))

	err = json.Unmarshal(res2, &aggReport)
	require.NoError(t, err)
	fmt.Println("Aggregate report: ", aggReport)

	require.Equal(t, aggReport.Aggregate.AggregateReporter, valAddress)

	// dispute report
	bz, err := json.Marshal(microReports.MicroReports[0])
	require.NoError(t, err)
	txHash, err = validatorI.ExecTx(ctx, "validator", "dispute", "propose-dispute", string(bz), "warning", "500000000000loya", "true", "--keyring-dir", layer.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)
	var disputes e2e.Disputes
	r, _, err := validatorI.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(t, err)
	err = json.Unmarshal(r, &disputes)
	require.NoError(t, err)
	require.Equal(t, disputes.Disputes[0].Metadata.DisputeStatus, 1) // voting
	fmt.Println("Disputes: ", string(r))
	res2, _, err = validatorI.ExecQuery(ctx, "oracle", "get-current-aggregate-report", hex.EncodeToString(qidbz))
	require.NoError(t, err)

	fmt.Println("Aggregate report: ", string(res2))
	// reporter should be jailed
	res3, _, err := validatorI.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(t, err)
	fmt.Println("Reporter: ", string(res3))
	// vote on dispute
	for _, v := range layer.Validators {
		_, err = v.ExecTx(ctx, "validator", "dispute", "vote", "1", "vote-support", "--keyring-dir", layer.HomeDir())
		require.NoError(t, err)
	}
	_, err = validatorI.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)

	res3, _, err = validatorI.ExecQuery(ctx, "dispute", "team-vote", "1")
	require.NoError(t, err)
	fmt.Println("Team address: ", string(res3))
	r, _, err = validatorI.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(t, err)

	err = json.Unmarshal(r, &disputes)
	require.NoError(t, err)
	require.Equal(t, disputes.Disputes[0].Metadata.DisputeStatus, 2) // resolved
}
