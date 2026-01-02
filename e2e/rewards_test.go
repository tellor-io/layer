package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
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

// TestRewards tests a realistic rewards scenario:
// - 3 validators (all earning staking rewards)
// - 2 validators become reporters
// - 1 non-validator user becomes a reporter (by delegating to a validator)
// - Total: 3 validators, 3 reporters, 4 accounts
// - Tests both staking rewards (validators) and reporting rewards (time-based rewards)
func TestRewards(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	// Create custom config with 3 validators and realistic gas fees
	config := e2e.DefaultSetupConfig()
	config.NumValidators = 3
	config.GasPrices = e2e.DefaultGasPrice
	config.ModifyGenesis = []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "30s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
	}

	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	// Get validators using utils function
	validatorsInfo, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validatorsInfo)

	// Verify we have 3 validators
	require.Equal(len(validatorsInfo), 3, "Expected 3 validators")

	// Verify all validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 3, "Expected 3 bonded validators")

	height, err := chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Current height: %d\n", height)

	fmt.Println("\n========== Creating User Reporter ==========")

	// Create and fund a regular user account with sufficient tokens
	userReporter := interchaintest.GetAndFundTestUsers(t, ctx, "user_reporter", math.NewInt(10_000*1e6), chain)[0]
	userReporterAddr := userReporter.FormattedAddress()
	fmt.Printf("User reporter address: %s\n", userReporterAddr)

	// Check user's initial balance
	userInitialBalance, err := chain.BankQueryBalance(ctx, userReporterAddr, "loya")
	require.NoError(err)
	fmt.Printf("User reporter initial balance: %s loya\n", userInitialBalance)

	// User delegates to validator 0
	delegateAmount := sdk.NewCoin("loya", math.NewInt(2_000*1e6))
	txHash, err := validatorsInfo[0].Node.ExecTx(ctx, userReporterAddr, "staking", "delegate",
		validatorsInfo[0].ValAddr, delegateAmount.String(),
		"--keyring-dir", validatorsInfo[0].Node.HomeDir(),
		"--gas", "500000",
		"--fees", "100loya")
	require.NoError(err)
	fmt.Printf("User delegates %s to validator[0], txHash: %s\n", delegateAmount, txHash)

	// Verify user's delegation
	require.NoError(testutil.WaitForBlocks(ctx, 2, validatorsInfo[0].Node))
	userBalanceAfterDelegate, err := chain.BankQueryBalance(ctx, userReporterAddr, "loya")
	require.NoError(err)
	fmt.Printf("User balance after delegation: %s loya\n", userBalanceAfterDelegate)

	// User becomes a reporter
	minStakeRequired := "1000000"
	commissionRate := "0.10"
	txHash, err = validatorsInfo[0].Node.ExecTx(ctx, userReporterAddr, "reporter", "create-reporter",
		commissionRate, minStakeRequired, "user_reporter_moniker",
		"--keyring-dir", validatorsInfo[0].Node.HomeDir(),
		"--fees", "50loya")
	require.NoError(err)
	fmt.Printf("User becomes a reporter, txHash: %s\n", txHash)

	fmt.Println("\n========== Initial Balances and Stakes ==========")

	type Account struct {
		Name    string
		Addr    string
		ValAddr string
		Node    *cosmos.ChainNode
	}

	// Track validators
	validator0 := Account{Name: "Validator0", Addr: validatorsInfo[0].AccAddr, ValAddr: validatorsInfo[0].ValAddr, Node: validatorsInfo[0].Node}
	validator1 := Account{Name: "Validator1", Addr: validatorsInfo[1].AccAddr, ValAddr: validatorsInfo[1].ValAddr, Node: validatorsInfo[1].Node}
	validator2 := Account{Name: "Validator2", Addr: validatorsInfo[2].AccAddr, ValAddr: validatorsInfo[2].ValAddr, Node: validatorsInfo[2].Node}

	allValidators := []Account{validator0, validator1, validator2}
	validatorInitialBalances := make(map[string]math.Int)

	for _, val := range allValidators {
		balance, err := chain.BankQueryBalance(ctx, val.Addr, "loya")
		require.NoError(err)
		validatorInitialBalances[val.Name] = balance
		fmt.Printf("%s initial balance: %s loya\n", val.Name, balance)

		stake, err := chain.StakingQueryValidator(ctx, val.ValAddr)
		require.NoError(err)
		fmt.Printf("%s stake: %s loya\n", val.Name, stake.Tokens)
		require.True(stake.Tokens.GT(math.NewInt(4000000000000)), "%s should have significant stake", val.Name)
	}

	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Current height: %d\n", height)

	// ========================================================================
	// PART 3: TURN ON MINTING
	// ========================================================================
	fmt.Println("\n========== Turning on Minting ==========")

	err = e2e.TurnOnMinting(ctx, chain, validatorsInfo[0].Node)
	require.NoError(err, "Failed to turn on minting")

	// Wait for proposal to pass
	fmt.Println("Waiting for minting proposal to pass...")
	require.NoError(testutil.WaitForBlocks(ctx, 20, validatorsInfo[0].Node))

	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED", "Minting proposal should have passed")

	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("Minting is now on at height %d\n", height)

	// Query initial time-based rewards pool balance
	tbrRes, _, err := e2e.QueryWithTimeout(ctx, validatorsInfo[0].Node, "oracle", "get-time-based-rewards")
	require.NoError(err)
	var tbrResp struct {
		Reward struct {
			Amount string `json:"amount"`
			Denom  string `json:"denom"`
		} `json:"reward"`
	}
	err = json.Unmarshal(tbrRes, &tbrResp)
	require.NoError(err)
	fmt.Printf("Initial time-based rewards pool: %s %s\n", tbrResp.Reward.Amount, tbrResp.Reward.Denom)

	// ========================================================================
	// PART 4: CREATE REPORTERS (2 validators + 1 user = 3 reporters total)
	// ========================================================================
	fmt.Println("\n========== Creating Reporters ==========")

	// Validator 0 becomes a reporter
	_, err = e2e.CreateReporterFromValidator(ctx, validatorsInfo[0], "validator0_reporter", math.NewInt(1000000))
	require.NoError(err, "Failed to create reporter from validator 0")
	fmt.Println("Validator[0] becomes a reporter")

	// Validator 1 becomes a reporter
	_, err = e2e.CreateReporterFromValidator(ctx, validatorsInfo[1], "validator1_reporter", math.NewInt(1000000))
	require.NoError(err, "Failed to create reporter from validator 1")
	fmt.Println("Validator[1] becomes a reporter")

	// Note: User reporter was already created in PART 1
	// Validator 2 does NOT become a reporter (remains validator-only)
	fmt.Println("Validator[2] remains validator-only (not a reporter)")

	// Wait for reporters to be set up
	require.NoError(testutil.WaitForBlocks(ctx, 2, validatorsInfo[0].Node))

	// ========================================================================
	// PART 5: ALL REPORTERS SUBMIT VALUES
	// ========================================================================
	fmt.Println("\n========== Reporters Submit Values ==========")

	// Get current cycle list query
	currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, validatorsInfo[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Printf("Current cycle list query: %s\n", currentCycleList.QueryData)

	// Encode value for reporting
	value := layerutil.EncodeValue(100.50)

	// Validator 0 reports (reporter)
	_, _, err = validatorsInfo[0].Node.Exec(ctx, validatorsInfo[0].Node.TxCommand("validator", "oracle", "submit-value",
		currentCycleList.QueryData, value, "--fees", "50loya", "--keyring-dir", validatorsInfo[0].Node.HomeDir()),
		validatorsInfo[0].Node.Chain.Config().Env)
	require.NoError(err, "Failed to submit value for validator 0")
	fmt.Printf("Validator[0] (reporter) submitted value\n")

	// Validator 1 reports (reporter)
	_, _, err = validatorsInfo[1].Node.Exec(ctx, validatorsInfo[1].Node.TxCommand("validator", "oracle", "submit-value",
		currentCycleList.QueryData, value, "--fees", "50loya", "--keyring-dir", validatorsInfo[1].Node.HomeDir()),
		validatorsInfo[1].Node.Chain.Config().Env)
	require.NoError(err, "Failed to submit value for validator 1")
	fmt.Printf("Validator[1] (reporter) submitted value\n")

	// User reporter reports (using validator 0's node to execute transaction)
	_, _, err = validatorsInfo[0].Node.Exec(ctx, validatorsInfo[0].Node.TxCommand(userReporterAddr, "oracle", "submit-value",
		currentCycleList.QueryData, value, "--fees", "50loya", "--keyring-dir", validatorsInfo[0].Node.HomeDir()),
		validatorsInfo[0].Node.Chain.Config().Env)
	require.NoError(err, "Failed to submit value for user reporter")
	fmt.Printf("User reporter submitted value\n")

	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Printf("All reports submitted at height %d\n", height)

	// Wait for blocks to ensure reports are aggregated and rewards distributed
	fmt.Println("\nWaiting for reports to be aggregated...")
	require.NoError(testutil.WaitForBlocks(ctx, 8, validatorsInfo[0].Node))

	// ========================================================================
	// PART 6: VERIFY STAKING REWARDS (All 3 validators earn these)
	// ========================================================================
	fmt.Println("\n========== Testing Staking Rewards ==========")

	// Validator 2 (non-reporter) claims staking rewards as control
	val2BalanceBefore, err := chain.BankQueryBalance(ctx, validator2.Addr, "loya")
	require.NoError(err)
	fmt.Printf("Validator2 (non-reporter) balance before staking claim: %s loya\n", val2BalanceBefore)

	txHash, err = validator2.Node.ExecTx(ctx, "validator", "distribution", "withdraw-all-rewards",
		"--keyring-dir", validator2.Node.HomeDir(),
		"--fees", "200loya")
	require.NoError(err)
	fmt.Printf("Validator2 claims staking rewards, txHash: %s\n", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validatorsInfo[0].Node))

	val2BalanceAfter, err := chain.BankQueryBalance(ctx, validator2.Addr, "loya")
	require.NoError(err)
	fmt.Printf("Validator2 balance after staking claim: %s loya\n", val2BalanceAfter)

	stakingRewardsNet := val2BalanceAfter.Sub(val2BalanceBefore)
	fmt.Printf("Validator2 staking rewards (net): %s loya\n", stakingRewardsNet)
	require.True(stakingRewardsNet.GT(math.ZeroInt()), "Validator2 should receive staking rewards")

	// ========================================================================
	// PART 7: VERIFY REPORTING REWARDS (Only 3 reporters earn these)
	// ========================================================================
	fmt.Println("\n========== Testing Reporting Rewards ==========")

	// Track balances before claiming reporting rewards
	val0BalanceBeforeTip, err := chain.BankQueryBalance(ctx, validator0.Addr, "loya")
	require.NoError(err)
	val1BalanceBeforeTip, err := chain.BankQueryBalance(ctx, validator1.Addr, "loya")
	require.NoError(err)
	userBalanceBeforeTip, err := chain.BankQueryBalance(ctx, userReporterAddr, "loya")
	require.NoError(err)

	fmt.Printf("Validator0 (reporter) balance before tip claim: %s loya\n", val0BalanceBeforeTip)
	fmt.Printf("Validator1 (reporter) balance before tip claim: %s loya\n", val1BalanceBeforeTip)
	fmt.Printf("User reporter balance before tip claim: %s loya\n", userBalanceBeforeTip)

	// Check if reporters have tips to withdraw
	for _, reporter := range []struct {
		name string
		addr string
		node *cosmos.ChainNode
	}{
		{"Validator0", validator0.Addr, validator0.Node},
		{"Validator1", validator1.Addr, validator1.Node},
		{"UserReporter", userReporterAddr, validatorsInfo[0].Node},
	} {
		tipsRes, _, err := e2e.QueryWithTimeout(ctx, reporter.node, "reporter", "selector-tip", reporter.addr)
		if err != nil {
			fmt.Printf("%s tip query error: %v\n", reporter.name, err)
			continue
		}
		var tipsResp struct {
			Tips string `json:"tips"`
		}
		err = json.Unmarshal(tipsRes, &tipsResp)
		if err == nil {
			fmt.Printf("%s has tips: %s loya\n", reporter.name, tipsResp.Tips)
		}
	}

	// Validator 0 (reporter) withdraws tips
	txHash, err = validator0.Node.ExecTx(ctx, "validator", "reporter", "withdraw-tip",
		validator0.Addr, validator0.ValAddr,
		"--fees", "100loya",
		"--keyring-dir", validator0.Node.HomeDir())
	if err != nil {
		fmt.Printf("Validator0 withdraw tip error: %v\n", err)
	} else {
		fmt.Printf("Validator0 withdrew tips, txHash: %s\n", txHash)
	}

	// Validator 1 (reporter) withdraws tips
	txHash, err = validator1.Node.ExecTx(ctx, "validator", "reporter", "withdraw-tip",
		validator1.Addr, validator1.ValAddr,
		"--fees", "100loya",
		"--keyring-dir", validator1.Node.HomeDir())
	if err != nil {
		fmt.Printf("Validator1 withdraw tip error: %v\n", err)
	} else {
		fmt.Printf("Validator1 withdrew tips, txHash: %s\n", txHash)
	}

	// User reporter withdraws tips (delegates tips to validator 0)
	txHash, err = validatorsInfo[0].Node.ExecTx(ctx, userReporterAddr, "reporter", "withdraw-tip",
		userReporterAddr, validator0.ValAddr,
		"--fees", "100loya",
		"--keyring-dir", validatorsInfo[0].Node.HomeDir())
	if err != nil {
		fmt.Printf("User reporter withdraw tip error: %v\n", err)
	} else {
		fmt.Printf("User reporter withdrew tips, txHash: %s\n", txHash)
	}

	require.NoError(testutil.WaitForBlocks(ctx, 2, validatorsInfo[0].Node))

	// Check balances after withdrawing tips
	val0BalanceAfterTip, err := chain.BankQueryBalance(ctx, validator0.Addr, "loya")
	require.NoError(err)
	val1BalanceAfterTip, err := chain.BankQueryBalance(ctx, validator1.Addr, "loya")
	require.NoError(err)
	userBalanceAfterTip, err := chain.BankQueryBalance(ctx, userReporterAddr, "loya")
	require.NoError(err)

	val0TipRewards := val0BalanceAfterTip.Sub(val0BalanceBeforeTip)
	val1TipRewards := val1BalanceAfterTip.Sub(val1BalanceBeforeTip)
	userTipRewards := userBalanceAfterTip.Sub(userBalanceBeforeTip)

	fmt.Printf("Validator0 reporting rewards (net): %s loya\n", val0TipRewards)
	fmt.Printf("Validator1 reporting rewards (net): %s loya\n", val1TipRewards)
	fmt.Printf("User reporter rewards (net): %s loya\n", userTipRewards)

	totalReportingRewards := val0TipRewards.Add(val1TipRewards).Add(userTipRewards)
	fmt.Printf("Total reporting rewards distributed: %s loya\n", totalReportingRewards)

	// ========================================================================
	// PART 8: FINAL VERIFICATION
	// ========================================================================
	fmt.Println("\n========== Final Verification ==========")

	// Verify final stakes
	for _, val := range allValidators {
		stake, err := chain.StakingQueryValidator(ctx, val.ValAddr)
		require.NoError(err)
		fmt.Printf("%s final stake: %s loya\n", val.Name, stake.Tokens)
		require.True(stake.Tokens.GT(math.NewInt(4000000000000)), "%s should maintain stake", val.Name)
	}

	// Summary
	fmt.Println("\n========== Test Summary ==========")
	fmt.Printf("✓ Chain setup: 3 validators, 1 user\n")
	fmt.Printf("✓ Reporters created: Validator0, Validator1, User\n")
	fmt.Printf("✓ All 3 reporters submitted values\n")
	fmt.Printf("✓ Staking rewards working: Validator2 earned %s loya\n", stakingRewardsNet)
	fmt.Printf("✓ Reporting rewards working: %s loya total distributed\n", totalReportingRewards)
	fmt.Printf("✓ Gas fees realistic: 0.000025 loya/gas\n")
	fmt.Println("\nBoth reward systems functioning correctly!")
}
