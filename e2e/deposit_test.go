package e2e_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// to run the test:
// in x/bridge/keeper/claim_deposit.go, change the 12 hr check to 2 min
// in x/registry/module/genesis.go, change the trbrbidge data spec ReportBlockWindow to 10
// in x/oracle/keeper/keeper.go, change the AutoClaimDeposit threshold to 2 min
// cd e2e
// go test -run TestDepositReport -timeout 10m
func TestDepositReport(t *testing.T) {
	t.Skip("change checks and run manually")

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	require := require.New(t)
	t.Parallel()

	// Use standard configuration
	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	// fileName := fmt.Sprintf("./dbfile/%s.db", t.Name())
	// Note: The standard setup already handles team key recovery and funding

	type Validators struct {
		Addr    string
		ValAddr string
		Val     *cosmos.ChainNode
	}

	validators := make([]Validators, len(chain.Validators))
	for i := range chain.Validators {
		val := chain.Validators[i]
		valAddr, err := val.AccountKeyBech32(ctx, "validator")
		require.NoError(err)
		valvalAddr, err := val.KeyBech32(ctx, "validator", "val")
		require.NoError(err)
		fmt.Println("val", i, " Account Address: ", valAddr)
		fmt.Println("val", i, " Validator Address: ", valvalAddr)
		validators[i] = Validators{
			Addr:    valAddr,
			ValAddr: valvalAddr,
			Val:     val,
		}
	}

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Val))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Val))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// both validators become reporters
	for i := range validators {
		minStakeAmt := "1000000"
		moniker := fmt.Sprintf("reporter_moniker%d", i)
		txHash, err := validators[i].Val.ExecTx(ctx, validators[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, moniker, "--keyring-dir", validators[i].Val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (validator", i, " becomes a reporter): ", txHash)
	}

	// validator tips bridge deposit id 1
	bridgeQueryDataString := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"
	tip := sdk.NewCoin("loya", math.NewInt(1*1e6))
	txHash, err := validators[0].Val.ExecTx(ctx, validators[0].Addr, "oracle", "tip", bridgeQueryDataString, tip.String(), "--keyring-dir", validators[0].Val.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val tips bridge deposit 1)", txHash)

	// both reporters report for the bridge deposit

	value := "0000000000000000000000003386518f7ab3eb51591571adbe62cf94540ead2900000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000000000000000174876e800000000000000000000000000000000000000000000000000000000000000002d74656c6c6f72317038386a7530796875746d6635703275373938787633756d616137756a77376763683972346600000000000000000000000000000000000000"
	for i := range validators {
		txHash, err := validators[i].Val.ExecTx(ctx, validators[i].Addr, "oracle", "submit-value", bridgeQueryDataString, value, "--keyring-dir", validators[0].Val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (validator", i, "reports bridge deposit 1)", txHash)
	}

	// make sure trbrbdige query is 10 blocks
	res, _, err := e2e.QueryWithTimeout(ctx, validators[0].Val, "registry", "data-spec", "TRBBridge")
	require.NoError(err)
	var specRes e2e.QueryGetDataSpecResponse
	err = json.Unmarshal(res, &specRes)
	require.NoError(err)
	require.NotNil(specRes)
	fmt.Println("spec res: ", specRes.Spec)

	// wait 10 blocks for aggregate
	require.NoError(testutil.WaitForBlocks(ctx, 10, validators[0].Val))

	// verify aggregate
	queryDataBz, err := hex.DecodeString(bridgeQueryDataString)
	require.NoError(err)
	queryIdBz := utils.QueryIDFromData(queryDataBz)
	queryIdString := hex.EncodeToString(queryIdBz)
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Val, "oracle", "get-current-aggregate-report", queryIdString)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	require.NotNil(currentAggRes)

	// check if deposit claimed
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Val, "bridge", "get-deposit-claimed", "1")
	require.NoError(err)
	var claimedRes e2e.QueryGetDepositClaimedResponse
	err = json.Unmarshal(res, &claimedRes)
	require.NoError(err)
	require.False(claimedRes.Claimed)

	loyaHolders, err := chain.BankQueryDenomOwners(ctx, "loya")
	require.NoError(err)
	fmt.Println("Loya holders: ", loyaHolders)
	fmt.Println("len(loyaHolders): ", len(loyaHolders))
	numHoldersBefore := len(loyaHolders)

	// wait for 2 min window to expire, deposit should get claimed
	time.Sleep(120 * time.Second)

	// check if deposit claimed
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Val, "bridge", "get-deposit-claimed", "1")
	require.NoError(err)
	err = json.Unmarshal(res, &claimedRes)
	require.NoError(err)
	require.True(claimedRes.Claimed)

	// check that there is a new loya holder
	loyaHolders, err = chain.BankQueryDenomOwners(ctx, "loya")
	require.NoError(err)
	fmt.Println("Loya holders: ", loyaHolders)
	fmt.Println("len(loyaHolders): ", len(loyaHolders))
	numHoldersAfter := len(loyaHolders)
	require.Greater(numHoldersAfter, numHoldersBefore)
}
