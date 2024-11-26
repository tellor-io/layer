package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"
)

const (
	bchusdQData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003626368000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	bchusdValue = "000000000000000000000000000000000000000000000058528649cf80ee0000" //

	ltcusdQData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000036c7463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	ltcusdQId   = "19585d912afb72378e3986a7a53f1eae1fbae792cd17e1d0df063681326823ae"
)

func TestLearn(t *testing.T) {
	ctx := context.Background()
	layer := e2e.LayerSpinup(t) // *cosmos.CosmosChain type
	validatorI := layer.Validators[0]

	valAddress, err := validatorI.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	// sample of how add a user and fund it
	user := interchaintest.GetAndFundTestUsers(t, ctx, "user1", math.OneInt(), layer)[0]
	fmt.Println("User address: ", user.FormattedAddress())

	prop := e2e.Proposal{
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
	_, err = e2e.ExecProposal(ctx, "validator", prop, validatorI)
	require.NoError(t, err)

	for _, v := range layer.Validators {
		_, err = v.ExecTx(ctx, "validator", "gov", "vote", "1", "yes", "--gas", "1000000", "--fees", "1000000loya", "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(t, err)
	}
	err = testutil.WaitForBlocks(ctx, 3, validatorI)
	require.NoError(t, err)

	result, err := layer.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	fmt.Println("Proposal result: ", result)
	// create reporter
	txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", valAddress, bchusdQData, "1000000loya", "--keyring-dir", "/var/cosmos-chain/layer-1"), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)

	txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valAddress, bchusdQData, bchusdValue, "--keyring-dir", "/var/cosmos-chain/layer-1")
	require.NoError(t, err)
	fmt.Println("Tx hash: ", txHash)

	res1, _, err := validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAddress)
	require.NoError(t, err)

	var microReports e2e.ReportsResponse
	err = json.Unmarshal(res1, &microReports)
	require.NoError(t, err)
	fmt.Println("Micro reports: ", microReports)
	require.Equal(t, microReports.MicroReports[0].Reporter, valAddress)

	// get aggreate report
	qidbz, err := utils.QueryIDFromDataString(bchusdQData)
	require.NoError(t, err)

	res2, _, err := validatorI.ExecQuery(ctx, "oracle", "get-current-aggregate-report", hex.EncodeToString(qidbz))
	require.NoError(t, err)

	var aggReport e2e.AggregateReport
	fmt.Println("Aggregate report: ", string(res2))

	err = json.Unmarshal(res2, &aggReport)
	require.NoError(t, err)
	fmt.Println("Aggregate report: ", aggReport)

	require.Equal(t, aggReport.Aggregate.AggregateReporter, valAddress)

	// dispute report
	bz, err := json.Marshal(microReports.MicroReports[0])
	require.NoError(t, err)
	txHash, err = validatorI.ExecTx(ctx, "validator", "dispute", "propose-dispute", string(bz), "warning", "500000000000loya", "true", "--keyring-dir", "/var/cosmos-chain/layer-1", "--gas", "1000000", "--fees", "1000000loya")
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
		_, err = v.ExecTx(ctx, "validator", "dispute", "vote", "1", "vote-support", "--keyring-dir", "/var/cosmos-chain/layer-1")
		require.NoError(t, err)
	}
	_, err = validatorI.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", "/var/cosmos-chain/layer-1")
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
