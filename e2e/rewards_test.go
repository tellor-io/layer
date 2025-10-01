package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestRewards --timeout 5m

// TestRewards tests the rewards system with 10 validators reporting and claiming rewards
func TestRewards(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	// Create custom config with 10 validators
	config := e2e.DefaultSetupConfig()
	config.NumValidators = 10
	config.ModifyGenesis = []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "60s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "20s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
		cosmos.NewGenesisKV("app_state.registry.dataspec.0.report_block_window", "5"),
	}

	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	// Get validators using utils function
	validatorsInfo, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validatorsInfo)

	// Verify we have 10 validators
	require.Equal(len(validatorsInfo), 10, "Expected 10 validators")

	type Validator struct {
		Addr                string
		ValAddr             string
		Val                 *cosmos.ChainNode
		FreeFloatingBalance string
		StakeAmount         string
	}

	validators := make([]Validator, len(validatorsInfo))
	initialBalances := make([]math.Int, len(validatorsInfo))

	for i, v := range validatorsInfo {
		freeFloatingBalance, err := chain.BankQueryBalance(ctx, v.AccAddr, "loya")
		require.NoError(err)
		fmt.Printf("validator [%d] free floating balance: %s\n", i, freeFloatingBalance)

		// Store initial balance for later comparison
		initialBalances[i] = freeFloatingBalance

		stakeAmount, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err)
		fmt.Printf("validator [%d] stake amount: %s\n", i, stakeAmount.Tokens)

		// Verify initial stake is reasonable (should be around 5B loya for validators)
		require.True(stakeAmount.Tokens.GT(math.NewInt(4000000000000)), "Validator %d should have significant stake", i)

		validators[i] = Validator{
			Addr:                v.AccAddr,
			ValAddr:             v.ValAddr,
			Val:                 v.Node,
			FreeFloatingBalance: freeFloatingBalance.String(),
			StakeAmount:         stakeAmount.Tokens.String(),
		}
	}

	// Verify all validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 10, "Expected 10 bonded validators")

	height, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Current height: %d\n", height)

	// Turn on minting using utils function
	err = e2e.TurnOnMinting(ctx, chain, validatorsInfo[0].Node)
	require.NoError(err, "Failed to turn on minting")

	// Wait for proposal to pass
	require.NoError(testutil.WaitForBlocks(ctx, 20, validatorsInfo[0].Node))

	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED", "Minting proposal should have passed")

	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Minting is now on at height %d\n", height)

	// Turn first validator into a reporter using utils function
	_, err = e2e.CreateReporterFromValidator(ctx, validatorsInfo[0], "val0_moniker", math.NewInt(1000000))
	require.NoError(err, "Failed to create reporter from validator 0")
	fmt.Println("validator [0] becomes a reporter")

	// Get current cycle list query
	currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, validatorsInfo[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Printf("Current cycle list: %+v\n", currentCycleList)

	// Encode value for reporting
	value := layerutil.EncodeValue(123456789.99)

	// First validator reports
	_, _, err = validatorsInfo[0].Node.Exec(ctx, validatorsInfo[0].Node.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "5loya", "--keyring-dir", validatorsInfo[0].Node.HomeDir()), validatorsInfo[0].Node.Chain.Config().Env)
	require.NoError(err, "Failed to submit value for validator 0")

	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Validator [0] reported at height %d\n", height)

	// Turn remaining validators into reporters
	for i := 1; i < len(validatorsInfo); i++ {
		moniker := fmt.Sprintf("val%d_moniker", i)
		_, err = e2e.CreateReporterFromValidator(ctx, validatorsInfo[i], moniker, math.NewInt(1000000))
		require.NoError(err, "Failed to create reporter from validator %d", i)
		fmt.Printf("Validator [%d] becomes a reporter\n", i)
	}

	// All validators report for the cycle list
	for i, v := range validatorsInfo {
		// Skip validator 0 as they already reported
		if i == 0 {
			continue
		}

		currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, v.Node, "oracle", "current-cyclelist-query")
		require.NoError(err, "Failed to query current cycle list for validator %d", i)

		var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
		err = json.Unmarshal(currentCycleListRes, &currentCycleList)
		require.NoError(err, "Failed to unmarshal current cycle list for validator %d", i)

		// Submit value for the cycle list
		_, _, err = v.Node.Exec(ctx, v.Node.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "5loya", "--keyring-dir", v.Node.HomeDir()), v.Node.Chain.Config().Env)
		require.NoError(err, "Failed to submit value for validator %d", i)

		height, err = chain.Height(ctx)
		require.NoError(err)
		fmt.Printf("Validator [%d] reported at height %d\n", i, height)
	}

	// Check validator 1 stake and balance before claim
	dels, err := chain.StakingQueryDelegations(ctx, validators[1].Addr)
	require.NoError(err, "Failed to query delegations for validator 1")
	fmt.Printf("VAL1 delegations before claim: %+v\n", dels)

	val1BalanceBefore, err := chain.BankQueryBalance(ctx, validators[1].Addr, "loya")
	require.NoError(err, "Failed to query balance for validator 1")
	fmt.Printf("VAL1 free floating balance before claim: %s\n", val1BalanceBefore)

	// Check validator 0 stake and balance before claim
	del, err := chain.StakingQueryDelegations(ctx, validators[0].Addr)
	require.NoError(err, "Failed to query delegations for validator 0")
	fmt.Printf("VAL0 delegations before claim: %+v\n", del)

	val0BalanceBefore, err := chain.BankQueryBalance(ctx, validators[0].Addr, "loya")
	require.NoError(err, "Failed to query balance for validator 0")
	fmt.Printf("VAL0 free floating balance before claim: %s\n", val0BalanceBefore)

	// Claim validator rewards (val1 pays for val0 to claim val0 rewards)
	txHash, err := validators[1].Val.ExecTx(ctx, "validator", "distribution", "withdraw-all-rewards", "--keyring-dir", validators[1].Val.HomeDir(), "--from", validators[0].Addr, "--fees", "2222loya")
	require.NoError(err, "Failed to claim validator rewards")
	fmt.Printf("TX HASH (val1 pays for val0 to claim val0 rewards): %s\n", txHash)

	// Check validator 1 stake and balance after claim
	dels, err = chain.StakingQueryDelegations(ctx, validators[1].Addr)
	require.NoError(err, "Failed to query delegations for validator 1 after claim")
	fmt.Printf("VAL1 delegations after claim: %+v\n", dels)

	val1BalanceAfter, err := chain.BankQueryBalance(ctx, validators[1].Addr, "loya")
	require.NoError(err, "Failed to query balance for validator 1 after claim")
	fmt.Printf("VAL1 free floating balance after claim: %s\n", val1BalanceAfter)

	// (val1 paid 2222 loya in fees)
	feePaid := val1BalanceBefore.Sub(val1BalanceAfter)
	require.True(feePaid.GTE(math.NewInt(2222)), "Val1 should have paid at least 2222 loya in fees")

	// Check validator 0 stake and balance after claim
	del, err = chain.StakingQueryDelegations(ctx, validators[0].Addr)
	require.NoError(err, "Failed to query delegations for validator 0 after claim")
	fmt.Printf("VAL0 delegations after claim: %+v\n", del)

	val0BalanceAfter, err := chain.BankQueryBalance(ctx, validators[0].Addr, "loya")
	require.NoError(err, "Failed to query balance for validator 0 after claim")
	fmt.Printf("VAL0 free floating balance after claim: %s\n", val0BalanceAfter)

	// Verify val0 balance increased due to claiming validator rewards
	require.True(val0BalanceAfter.GT(val0BalanceBefore), "Val0 balance should increase after claiming validator rewards")
	rewardsClaimed := val0BalanceAfter.Sub(val0BalanceBefore)
	fmt.Printf("Val0 claimed %s loya in validator rewards\n", rewardsClaimed)
	require.True(rewardsClaimed.GT(math.ZeroInt()), "Val0 should have received some validator rewards")

	// Wait for blocks to ensure all reports are processed
	require.NoError(testutil.WaitForBlocks(ctx, 6, validatorsInfo[0].Node))

	// Check reports per reporter
	for i, v := range validatorsInfo {
		reports, _, err := e2e.QueryWithTimeout(ctx, v.Node, "oracle", "get-reportsby-reporter", v.AccAddr, "--page-limit", "1")
		require.NoError(err, "Failed to query reports for validator %d", i)

		var reportsRes e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reports, &reportsRes)
		require.NoError(err, "Failed to unmarshal reports for validator %d", i)
		fmt.Printf("Reports from %s: %+v\n", v.AccAddr, reportsRes)
	}

	// Check each reporter's free floating balance and stake amount before claiming reporting rewards
	balancesBeforeReportingClaim := make([]math.Int, len(validatorsInfo))
	for i, v := range validatorsInfo {
		freeFloatingBalance, err := chain.BankQueryBalance(ctx, v.AccAddr, "loya")
		require.NoError(err, "Failed to query balance for validator %d", i)
		fmt.Printf("Validator [%s] free floating balance before reporting claim: %s\n", v.AccAddr, freeFloatingBalance)

		// Store balance for comparison after claiming
		balancesBeforeReportingClaim[i] = freeFloatingBalance

		stakeAmount, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err, "Failed to query stake for validator %d", i)
		fmt.Printf("Validator [%s] stake amount: %s\n", v.AccAddr, stakeAmount.Tokens)
	}

	// Claim reporting rewards for each validator/reporter
	successfulClaims := 0
	for i, v := range validatorsInfo {
		_, err = v.Node.ExecTx(ctx, "validator", "reporter", "withdraw-tip", v.AccAddr, v.ValAddr, "--fees", "5loya", "--keyring-dir", v.Node.HomeDir())
		if err != nil {
			fmt.Printf("Error from claiming rewards for validator %d (some people didn't get their report in): %v\n", i, err)
		} else {
			fmt.Printf("Validator [%s] claimed rewards successfully\n", v.AccAddr)
			successfulClaims++
		}
	}

	// Verify that at least some validators successfully claimed reporting rewards
	require.True(successfulClaims > 0, "At least one validator should have successfully claimed reporting rewards")
	fmt.Printf("Successfully claimed reporting rewards for %d out of %d validators\n", successfulClaims, len(validatorsInfo))

	// Final check on each validator/reporter's free floating balance and stake amount
	height, err = chain.Height(ctx)
	require.NoError(err, "Failed to get final height")
	fmt.Printf("Final height: %d\n", height)

	totalRewardsClaimed := math.ZeroInt()
	validatorsWithIncreasedBalance := 0

	for i, v := range validatorsInfo {
		finalBalance, err := chain.BankQueryBalance(ctx, v.AccAddr, "loya")
		require.NoError(err, "Failed to query final balance for validator %d", i)
		fmt.Printf("Validator [%s] final free floating balance: %s\n", v.AccAddr, finalBalance)

		// Compare with initial balance to see total rewards earned
		initialBalance := initialBalances[i]
		balanceChange := finalBalance.Sub(initialBalance)
		fmt.Printf("Validator [%s] total balance change from start: %s\n", v.AccAddr, balanceChange)

		// Check if balance increased from reporting rewards claim
		balanceBeforeReportingClaim := balancesBeforeReportingClaim[i]
		reportingRewardChange := finalBalance.Sub(balanceBeforeReportingClaim)
		if reportingRewardChange.GT(math.ZeroInt()) {
			validatorsWithIncreasedBalance++
			totalRewardsClaimed = totalRewardsClaimed.Add(reportingRewardChange)
			fmt.Printf("Validator [%s] gained %s from reporting rewards\n", v.AccAddr, reportingRewardChange)
		}

		stakeAmount, err := chain.StakingQueryValidator(ctx, v.ValAddr)
		require.NoError(err, "Failed to query final stake for validator %d", i)
		fmt.Printf("Validator [%s] final stake amount: %s\n", v.AccAddr, stakeAmount.Tokens)

		// Verify stake amount hasn't decreased (should remain stable or increase)
		require.True(stakeAmount.Tokens.GTE(math.NewInt(4000000000000)), "Validator %d should maintain significant stake", i)
	}

	// Verify that the rewards system is working
	require.True(validatorsWithIncreasedBalance > 0, "At least one validator should have increased balance from reporting rewards")
	require.True(totalRewardsClaimed.GT(math.ZeroInt()), "Total reporting rewards claimed should be positive")
	fmt.Printf("Total reporting rewards claimed across all validators: %s loya\n", totalRewardsClaimed)
	fmt.Printf("Validators with increased balance from reporting rewards: %d out of %d\n", validatorsWithIncreasedBalance, len(validatorsInfo))
}
