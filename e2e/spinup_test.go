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
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	qData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003626368000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value = "000000000000000000000000000000000000000000000058528649cf80ee0000"
)

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

func TestLayerFlow(t *testing.T) {
	ctx := context.Background()
	layer := e2e.LayerSpinup(t)
	validatorI := layer.Validators[0]
	validatorII := layer.Validators[1]

	valAddress, err := validatorI.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	valIIAddress, err := validatorII.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	// sample of how add a user and fund it
	user := interchaintest.GetAndFundTestUsers(t, ctx, "user1", math.OneInt(), layer)[0]
	fmt.Println("User address: ", user.FormattedAddress())

	disputer := interchaintest.GetAndFundTestUsers(t, ctx, "disputer", math.NewInt(1*1e12), layer)[0]
	disputerFA := disputer.FormattedAddress()

	// Test sending funds to extra_rewards_pool module account
	testFunder := interchaintest.GetAndFundTestUsers(t, ctx, "testFunder", math.NewInt(1_000_000), layer)[0]
	type Balance struct {
		Amount string `json:"amount"`
		Denom  string `json:"denom"`
	}
	type Response struct {
		AccountBalance Balance `json:"balance"`
	}
	// Test the module account can receive funds from an external account
	extraRewardsAddr := authtypes.NewModuleAddress("extra_rewards_pool").String()
	// check initial balance is zero
	balanceRes, _, err := e2e.QueryWithTimeout(ctx, validatorI, "bank", "balance", extraRewardsAddr, "loya")
	require.NoError(t, err)
	var balance Response
	err = json.Unmarshal(balanceRes, &balance)
	require.NoError(t, err)
	require.Equal(t, "0", balance.AccountBalance.Amount)
	// send funds to extra rewards pool
	_, err = validatorI.ExecTx(ctx, testFunder.KeyName(), "bank", "send", testFunder.FormattedAddress(), extraRewardsAddr, "100000loya", "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)

	// Verify the funds were received
	balanceRes, _, err = e2e.QueryWithTimeout(ctx, validatorI, "bank", "balance", extraRewardsAddr, "loya")
	require.NoError(t, err)
	err = json.Unmarshal(balanceRes, &balance)
	require.NoError(t, err)
	// convert to math.Int for comparison
	balInt, ok := math.NewIntFromString(balance.AccountBalance.Amount)
	require.True(t, ok)
	require.True(t, balInt.GT(math.ZeroInt()))

	// turn on minting
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
	// all validators vote yes on minting proposal
	for _, v := range layer.Validators {
		_, err = v.ExecTx(ctx, "validator", "gov", "vote", "1", "yes", "--gas", "1000000", "--fees", "500loya", "--keyring-dir", layer.HomeDir())
		require.NoError(t, err)
	}
	err = testutil.WaitForBlocks(ctx, 3, validatorI)
	require.NoError(t, err)

	result, err := layer.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	fmt.Println("Proposal result: ", result)

	// all validators become reporters
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val1_moniker", "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)
	txHash, err = validatorII.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val2_moniker", "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	// validatorI tips
	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", layer.HomeDir()), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)
	// query tippped queries
	res, _, err := e2e.QueryWithTimeout(ctx, validatorI, "oracle", "get-tipped-queries")
	require.NoError(t, err)
	var tippedQueries e2e.QueryGetTippedQueriesResponse
	err = json.Unmarshal(res, &tippedQueries)
	require.NoError(t, err)
	fmt.Println("Tipped queries: ", tippedQueries.ActiveQueries[0])
	// validatorI reports
	txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	// validatorII tips
	_, _, err = validatorII.Exec(ctx, validatorII.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", layer.HomeDir()), validatorII.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorII)
	require.NoError(t, err)
	// validatorII reports
	txHash, err = validatorII.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", layer.HomeDir())
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	res1, _, err := e2e.QueryWithTimeout(ctx, validatorI, "oracle", "get-reportsby-reporter", valAddress, "--page-limit", "1")
	require.NoError(t, err)

	var microReports e2e.ReportsResponse
	err = json.Unmarshal(res1, &microReports)
	require.NoError(t, err)
	fmt.Println("Micro reports: ", microReports)
	require.Equal(t, microReports.MicroReports[0].Reporter, valAddress)

	// get aggreate report
	qidbz, err := utils.QueryIDFromDataString(qData)
	require.NoError(t, err)

	res2, _, err := e2e.QueryWithTimeout(ctx, validatorI, "oracle", "get-current-aggregate-report", hex.EncodeToString(qidbz))
	require.NoError(t, err)

	var aggReport e2e.AggregateReport
	fmt.Println("Aggregate report: ", string(res2))

	err = json.Unmarshal(res2, &aggReport)
	require.NoError(t, err)
	fmt.Println("Aggregate report: ", aggReport)

	require.Equal(t, aggReport.Aggregate.AggregateReporter, valIIAddress)

	// second party disputes report
	txHash, err = validatorI.ExecTx(ctx, disputerFA, "dispute", "propose-dispute", microReports.MicroReports[0].Reporter, microReports.MicroReports[0].MetaId, microReports.MicroReports[0].QueryID, "warning", "500000000000loya", "false", "--keyring-dir", layer.HomeDir(), "--gas", "1000000", "--fees", "500loya")
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)
	var disputes e2e.Disputes
	r, _, err := e2e.QueryWithTimeout(ctx, validatorI, "dispute", "disputes")
	require.NoError(t, err)
	err = json.Unmarshal(r, &disputes)
	require.NoError(t, err)
	require.Equal(t, disputes.Disputes[0].Metadata.DisputeStatus, "DISPUTE_STATUS_VOTING") // voting
	fmt.Println("Disputes: ", string(r))
	res2, _, err = e2e.QueryWithTimeout(ctx, validatorI, "oracle", "get-current-aggregate-report", hex.EncodeToString(qidbz))
	require.NoError(t, err)

	fmt.Println("Aggregate report: ", string(res2))
	// reporter should be jailed
	res3, _, err := e2e.QueryWithTimeout(ctx, validatorI, "reporter", "reporters")
	require.NoError(t, err)
	fmt.Println("Reporter: ", string(res3))

	// validators(reporters and tippers) vote on dispute
	for _, v := range layer.Validators {
		_, err = v.ExecTx(ctx, "validator", "dispute", "vote", "1", "vote-support", "--keyring-dir", layer.HomeDir())
		require.NoError(t, err)
	}

	// check dispute status
	r, _, err = e2e.QueryWithTimeout(ctx, validatorI, "dispute", "disputes")
	require.NoError(t, err)

	err = json.Unmarshal(r, &disputes)
	require.NoError(t, err)
	require.Equal(t, disputes.Disputes[0].Metadata.DisputeStatus, "DISPUTE_STATUS_RESOLVED") // 2/3 voted so resolved

	// team votes should error
	_, err = validatorI.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", layer.HomeDir())
	require.Error(t, err) // vote already tallied

	r, _, err = e2e.QueryWithTimeout(ctx, validatorI, "dispute", "disputes")
	require.NoError(t, err)

	err = json.Unmarshal(r, &disputes)
	require.NoError(t, err)
	require.Equal(t, disputes.Disputes[0].Metadata.DisputeStatus, "DISPUTE_STATUS_RESOLVED") // resolved
}

func TestGetCyclelist(t *testing.T) {
	t.Skip("infinitely runs and calls CurrentCyclistQuery")
	ctx := context.Background()
	layer := e2e.LayerSpinup(t) // *cosmos.CosmosChain type
	validatorI := layer.Validators[0]

	queryClient := oracletypes.NewQueryClient(validatorI.GrpcConn)
	for {
		res, err := queryClient.CurrentCyclelistQuery(ctx, &oracletypes.QueryCurrentCyclelistQueryRequest{})
		require.NoError(t, err)
		// fmt.Println(res.QueryMeta)
		require.NotNil(t, res.QueryData)
		require.NotNil(t, res.QueryMeta)
	}
}
