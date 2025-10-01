package e2e_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestSelectorCreatesReporter(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	// Use standard configuration
	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	// Get validators
	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// create user selector
	fundAmt := math.NewInt(1_100 * 1e6)
	delegateAmt := sdk.NewCoin("loya", math.NewInt(1000*1e6)) // all tokens after paying fee
	user := interchaintest.GetAndFundTestUsers(t, ctx, "user", fundAmt, chain)[0]
	txHash, err := validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[1].ValAddr, delegateAmt.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", "10loya")
	require.NoError(err)
	fmt.Println("TX HASH (user delegates to val1): ", txHash)

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// query delegators to val1
	delegators, err := chain.StakingQueryDelegationsTo(ctx, validators[1].ValAddr)
	require.NoError(err)
	fmt.Println("delegators to val1: ", delegators)
	require.Equal(len(delegators), 2) // self delegation and user delegation

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Node))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// both validators become reporters
	for i := range validators {
		minStakeAmt := "1000000"
		moniker := fmt.Sprintf("reporter_moniker%d", i)
		txHash, err := validators[i].Node.ExecTx(ctx, validators[i].AccAddr, "reporter", "create-reporter", commissRate, minStakeAmt, moniker, "--keyring-dir", validators[i].Node.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (validator", i, " becomes a reporter): ", txHash)
	}

	// user selects val 1 as their reporter
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", validators[1].AccAddr, "--keyring-dir", validators[0].Node.HomeDir(), "--fees", "5loya")
	require.NoError(err)
	fmt.Println("TX HASH (user selects val1 as their reporter): ", txHash)

	//  both reporters submit for cyclelist
	currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Println("current cycle list: ", currentCycleList)
	value := layerutil.EncodeValue(123456789.99)
	for i := range validators {
		_, _, err = validators[i].Node.Exec(ctx, validators[i].Node.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "5loya", "--keyring-dir", validators[i].Node.HomeDir()), validators[i].Node.Chain.Config().Env)
		require.NoError(err)
		height, err := validators[i].Node.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}

	// wait for aggregation to complete
	require.NoError(testutil.WaitForBlocks(ctx, 6, validators[0].Node))

	// query report info
	qDataBz, err := hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz := utils.QueryIDFromData(qDataBz)
	qId := hex.EncodeToString(qIdBz)
	res, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	fmt.Println("current aggregate report power: ", currentAggRes.Aggregate.AggregatePower)
	report1Power, err := strconv.ParseUint(currentAggRes.Aggregate.AggregatePower, 10, 64)
	require.NoError(err)

	// user creates a reporter
	minStakeAmt := "1000000"
	moniker := "reporter_moniker"
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "create-reporter", commissRate, minStakeAmt, moniker, "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user creates a reporter): ", txHash)

	// all 3 reporters report
	currentCycleListRes, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	for i := range validators {
		_, _, err = validators[i].Node.Exec(ctx, validators[i].Node.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "5loya", "--keyring-dir", validators[i].Node.HomeDir()), validators[i].Node.Chain.Config().Env)
		require.NoError(err)
		height, err := validators[i].Node.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}
	_, _, err = validators[0].Node.Exec(ctx, validators[0].Node.TxCommand(user.FormattedAddress(), "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "5loya", "--keyring-dir", validators[0].Node.HomeDir()), validators[0].Node.Chain.Config().Env)
	require.NoError(err)

	// wait for aggregation to complete (need to wait for report_block_window)
	require.NoError(testutil.WaitForBlocks(ctx, 6, validators[0].Node))

	// query report info
	qDataBz, err = hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz = utils.QueryIDFromData(qDataBz)
	qId = hex.EncodeToString(qIdBz)
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	fmt.Println("current aggregate report power: ", currentAggRes.Aggregate.AggregatePower)
	report2Power, err := strconv.ParseUint(currentAggRes.Aggregate.AggregatePower, 10, 64)
	require.NoError(err)
	require.Greater(report2Power, report1Power) // report 2 should have more power than report 1 since we added a third reporter
}
