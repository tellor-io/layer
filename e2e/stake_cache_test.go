package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	stakeCacheCommissRate = "0.1"
	moniker               = "reporter_0"
	stakeCacheTxFee       = "500loya"
)

func stakeCacheSetupConfig() e2e.SetupConfig {
	config := e2e.DefaultSetupConfig()
	config.ModifyGenesis = append(config.ModifyGenesis,
		cosmos.NewGenesisKV("app_state.registry.dataspec.0.report_block_window", "10"),
	)
	return config
}

func waitForStakeCacheCycleListQuery(t *testing.T, ctx context.Context, node *cosmos.ChainNode, previousQueryMetaID string) e2e.QueryCurrentCyclelistQueryResponse {
	t.Helper()

	const minReportBlocksRemaining = 4

	for range 20 {
		currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, node, "oracle", "current-cyclelist-query")
		require.NoError(t, err)

		var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
		require.NoError(t, json.Unmarshal(currentCycleListRes, &currentCycleList))
		require.NotNil(t, currentCycleList.QueryMeta)

		if previousQueryMetaID != "" && currentCycleList.QueryMeta.Id == previousQueryMetaID {
			require.NoError(t, testutil.WaitForBlocks(ctx, 1, node))
			continue
		}

		expiration, err := strconv.ParseUint(currentCycleList.QueryMeta.Expiration, 10, 64)
		require.NoError(t, err)

		height, err := node.Height(ctx)
		require.NoError(t, err)
		if expiration >= uint64(height)+minReportBlocksRemaining {
			return currentCycleList
		}

		require.NoError(t, testutil.WaitForBlocks(ctx, 1, node))
	}

	require.FailNow(t, "timed out waiting for cyclelist query with enough remaining report blocks")
	return e2e.QueryCurrentCyclelistQueryResponse{}
}

func submitStakeCacheValue(t *testing.T, ctx context.Context, node *cosmos.ChainNode, queryData, value string) string {
	t.Helper()

	txBytes, _, err := node.Exec(ctx, node.TxCommand("validator", "oracle", "submit-value", queryData, value, "--fees", stakeCacheTxFee, "--keyring-dir", node.HomeDir()), node.Chain.Config().Env)
	require.NoError(t, err)
	txHash, err := e2e.GetTxHashFromExec(txBytes)
	require.NoError(t, err)
	return txHash
}

func waitForStakeCacheAggregatePower(t *testing.T, ctx context.Context, node *cosmos.ChainNode, queryData, expectedMetaID string) uint64 {
	t.Helper()

	qDataBz, err := hex.DecodeString(queryData)
	require.NoError(t, err)
	qId := hex.EncodeToString(utils.QueryIDFromData(qDataBz))

	var lastErr error
	for range 20 {
		res, _, err := e2e.QueryWithTimeout(ctx, node, "oracle", "get-current-aggregate-report", qId)
		if err != nil {
			lastErr = err
			require.NoError(t, testutil.WaitForBlocks(ctx, 1, node))
			continue
		}

		var aggRes e2e.QueryGetCurrentAggregateReportResponse
		require.NoError(t, json.Unmarshal(res, &aggRes))
		if aggRes.Aggregate == nil {
			lastErr = fmt.Errorf("aggregate response missing aggregate")
			require.NoError(t, testutil.WaitForBlocks(ctx, 1, node))
			continue
		}
		if aggRes.Aggregate.MetaId != expectedMetaID {
			lastErr = fmt.Errorf("latest aggregate meta id %s does not match expected meta id %s", aggRes.Aggregate.MetaId, expectedMetaID)
			require.NoError(t, testutil.WaitForBlocks(ctx, 1, node))
			continue
		}

		power, err := strconv.ParseUint(aggRes.Aggregate.AggregatePower, 10, 64)
		require.NoError(t, err)
		return power
	}

	require.FailNow(t, fmt.Sprintf("timed out waiting for aggregate report for query id %s: %v", qId, lastErr))
	return 0
}

// TestStakingHooksTriggered specifically tests that the reporter module's staking hooks
// fire when a delegation occurs for a registered selector.
// This isolates the hook behavior: if hooks fire, StakeRecalcFlag gets set,
// and the next report recalculates. If hooks don't fire, the cached (stale) stake is used.
func TestStakingHooksTriggered(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, stakeCacheSetupConfig())
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// Turn on minting
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Node))

	// Step 1: Validator 0 becomes a reporter
	minStakeAmt := "1000000"
	txHash, err := validators[0].Node.ExecTx(ctx, validators[0].AccAddr, "reporter", "create-reporter", stakeCacheCommissRate, minStakeAmt, moniker, "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("=== Step 1: Reporter created, txHash:", txHash)

	// Step 2: Create selector - fund, delegate, and join reporter
	fundAmt := math.NewInt(20_000 * 1e6)
	initialDelegate := sdk.NewCoin("loya", math.NewInt(1000*1e6))
	user := interchaintest.GetAndFundTestUsers(t, ctx, "selector", fundAmt, chain)[0]
	fmt.Println("=== Step 2: Selector address:", user.FormattedAddress())

	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[0].ValAddr, initialDelegate.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("=== Step 2: Initial delegation txHash:", txHash)
	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Verify delegation exists
	delRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "staking", "delegation", user.FormattedAddress(), validators[0].ValAddr)
	require.NoError(err)
	fmt.Println("=== Step 2: Delegation exists:", string(delRes))

	// Selector joins reporter
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", validators[0].AccAddr, "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("=== Step 2: Select-reporter txHash:", txHash)
	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Verify selector->reporter relationship
	selectorRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "reporter", "selector-reporter", user.FormattedAddress())
	require.NoError(err)
	var selectorReporter e2e.QuerySelectorReporterResponse
	require.NoError(json.Unmarshal(selectorRes, &selectorReporter))
	fmt.Println("=== Step 2: Selector's reporter:", selectorReporter.Reporter)
	require.Equal(validators[0].AccAddr, selectorReporter.Reporter, "Selector should be linked to validator 0's reporter")

	// Step 3: Submit first report to establish cached stake
	currentCycleList := waitForStakeCacheCycleListQuery(t, ctx, validators[0].Node, "")
	firstQueryMetaID := currentCycleList.QueryMeta.Id

	value := layerutil.EncodeValue(500.0)
	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Get first report power
	firstPower := waitForStakeCacheAggregatePower(t, ctx, validators[0].Node, currentCycleList.QueryData, currentCycleList.QueryMeta.Id)
	fmt.Println("=== Step 3: First report power (cached baseline):", firstPower)

	// Step 4: Selector delegates MORE - this MUST trigger AfterDelegationModified hook
	// The hook should set StakeRecalcFlag for the reporter
	additionalDelegate := sdk.NewCoin("loya", math.NewInt(10_000*1e6))
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[0].ValAddr, additionalDelegate.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("=== Step 4: Additional delegation txHash:", txHash)

	// Verify tx succeeded
	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))
	txRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "tx", txHash)
	require.NoError(err)
	fmt.Println("=== Step 4: Delegate TX result:", string(txRes))

	// Verify the delegation amount increased
	delRes2, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "staking", "delegation", user.FormattedAddress(), validators[0].ValAddr)
	require.NoError(err)
	fmt.Println("=== Step 4: Updated delegation:", string(delRes2))

	require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Node))

	// Step 5: Submit second report - if hooks fired, StakeRecalcFlag is set,
	// and ReporterStake will recalculate instead of using cache
	currentCycleList = waitForStakeCacheCycleListQuery(t, ctx, validators[0].Node, firstQueryMetaID)

	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Get second report power
	secondPower := waitForStakeCacheAggregatePower(t, ctx, validators[0].Node, currentCycleList.QueryData, currentCycleList.QueryMeta.Id)
	fmt.Println("=== Step 5: Second report power (after hook should recalc):", secondPower)

	// Step 6: Assertions
	// If AfterDelegationModified hook fired:
	//   - StakeRecalcFlag was set for the reporter
	//   - Second submit called ReporterStake which saw the flag
	//   - ReporterStake recalculated and includes the new delegation
	//   - Power should increase by ~10000 (10000 * 1e6 loya / 1e6 power reduction)
	// If hook did NOT fire:
	//   - No StakeRecalcFlag set
	//   - Second submit used cached stake (unless valset update also triggered recalc)
	//   - Power might still increase due to valset update recalc path
	expectedIncrease := uint64(10_000) // 10K loya in power units
	actualIncrease := secondPower - firstPower
	fmt.Printf("=== Step 6: Power increase: %d (expected ~%d)\n", actualIncrease, expectedIncrease)
	fmt.Printf("=== Step 6: First power: %d, Second power: %d\n", firstPower, secondPower)

	require.Greater(secondPower, firstPower, "Hook FAILED: Reporter stake should increase after selector's additional delegation. This means AfterDelegationModified hook did not fire or StakeRecalcFlag was not set.")
}

// TestStakeCacheValSetUpdate tests that reporter stake is recalculated after validator set update
// Scenario: Reporter submits, validator power changes, reporter submits again
// Expected: Second submission should recalculate stake with new validator power
func TestStakeCacheValSetUpdate(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	config := stakeCacheSetupConfig()
	config.NumValidators = 4
	config.ModifyGenesis = append(config.ModifyGenesis,
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "30s"),
	)
	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// Turn on minting
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Node))

	// Validator 0 becomes a reporter
	minStakeAmt := "1000000"
	txHash, err := validators[0].Node.ExecTx(ctx, validators[0].AccAddr, "reporter", "create-reporter", stakeCacheCommissRate, minStakeAmt, moniker, "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (validator 0 becomes reporter):", txHash)

	// Get current cyclelist query
	currentCycleList := waitForStakeCacheCycleListQuery(t, ctx, validators[0].Node, "")
	firstQueryMetaID := currentCycleList.QueryMeta.Id

	// First report
	value := layerutil.EncodeValue(100.0)
	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	fmt.Println("First report submitted")

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Query first report power
	firstPower := waitForStakeCacheAggregatePower(t, ctx, validators[0].Node, currentCycleList.QueryData, currentCycleList.QueryMeta.Id)
	fmt.Println("First report power:", firstPower)

	// Fund validator 0 with extra tokens and self-delegate to increase their own reporter stake.
	// The validator is already a selector of themselves (from create-reporter), so this delegation
	// triggers AfterDelegationModified hook and should be reflected in the next report's power.
	delegateAmt := sdk.NewCoin("loya", math.NewInt(1_000*1e6))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: validators[0].AccAddr,
		Amount:  math.NewInt(3_000_000 * 1e6),
		Denom:   "loya",
	}))
	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	txHash, err = validators[0].Node.ExecTx(ctx, validators[0].AccAddr, "staking", "delegate", validators[0].ValAddr, delegateAmt.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--gas", "500000", "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (validator 0 self-delegates more):", txHash)

	tooMuchSelfDelegate := sdk.NewCoin("loya", math.NewInt(2_000_000*1e6))
	_, err = validators[0].Node.ExecTx(ctx, validators[0].AccAddr, "staking", "delegate", validators[0].ValAddr, tooMuchSelfDelegate.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--gas", "500000", "--fees", stakeCacheTxFee)
	require.Error(err)
	require.ErrorContains(err, "total stake increase exceeds the allowed 5% threshold within a twelve-hour period")

	overLimitDelegator := interchaintest.GetAndFundTestUsers(t, ctx, "stake-share-over-limit", math.NewInt(10_000_000*1e6), chain)[0]
	tooMuchDelegatorStake := sdk.NewCoin("loya", math.NewInt(9_000_000*1e6))
	_, err = validators[0].Node.ExecTx(ctx, overLimitDelegator.FormattedAddress(), "staking", "delegate", validators[0].ValAddr, tooMuchDelegatorStake.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--gas", "500000", "--fees", stakeCacheTxFee)
	require.Error(err)
	require.ErrorContains(err, "total stake increase exceeds the allowed 5% threshold within a twelve-hour period")

	require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Node))

	// Second report - should trigger recalculation due to validator set update
	currentCycleList = waitForStakeCacheCycleListQuery(t, ctx, validators[0].Node, firstQueryMetaID)

	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	fmt.Println("Second report submitted")

	require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Node))

	// Query second report power
	secondPower := waitForStakeCacheAggregatePower(t, ctx, validators[0].Node, currentCycleList.QueryData, currentCycleList.QueryMeta.Id)
	fmt.Println("Second report power:", secondPower)

	// Second power should be greater due to new delegation
	require.Greater(secondPower, firstPower, "Reporter stake should increase after delegation")
}

// TestStakeCacheSelectorJoin tests that reporter stake is recalculated when a new selector joins
// Scenario: Reporter submits, new selector joins reporter, reporter submits again
// Expected: Second submission should include new selector's stake
func TestStakeCacheSelectorJoin(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// Turn on minting
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Node))

	// Create a user with stake
	fundAmt := math.NewInt(2_000 * 1e6)
	delegateAmt := sdk.NewCoin("loya", math.NewInt(1000*1e6))
	user := interchaintest.GetAndFundTestUsers(t, ctx, "selector", fundAmt, chain)[0]

	// User delegates to validator 1
	txHash, err := validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[1].ValAddr, delegateAmt.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user delegates):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Validator 0 becomes a reporter
	minStakeAmt := "1000000"
	txHash, err = validators[0].Node.ExecTx(ctx, validators[0].AccAddr, "reporter", "create-reporter", stakeCacheCommissRate, minStakeAmt, moniker, "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (validator 0 becomes reporter):", txHash)

	// First report (only validator's self-delegation)
	currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	require.NoError(json.Unmarshal(currentCycleListRes, &currentCycleList))

	value := layerutil.EncodeValue(200.0)
	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	fmt.Println("First report submitted")

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Query first report power
	qDataBz, err := hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz := utils.QueryIDFromData(qDataBz)
	qId := hex.EncodeToString(qIdBz)
	res, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	var aggRes e2e.QueryGetCurrentAggregateReportResponse
	require.NoError(json.Unmarshal(res, &aggRes))
	firstPower, err := strconv.ParseUint(aggRes.Aggregate.AggregatePower, 10, 64)
	require.NoError(err)
	fmt.Println("First report power:", firstPower)

	// User selects validator 0 as their reporter
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", validators[0].AccAddr, "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user selects reporter):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Second report - should include selector's stake
	currentCycleListRes, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	require.NoError(json.Unmarshal(currentCycleListRes, &currentCycleList))

	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	fmt.Println("Second report submitted")

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Query second report power
	qDataBz, err = hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz = utils.QueryIDFromData(qDataBz)
	qId = hex.EncodeToString(qIdBz)
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	require.NoError(json.Unmarshal(res, &aggRes))
	secondPower, err := strconv.ParseUint(aggRes.Aggregate.AggregatePower, 10, 64)
	require.NoError(err)
	fmt.Println("Second report power:", secondPower)

	// Second power should be greater due to new selector
	require.Greater(secondPower, firstPower, "Reporter stake should increase after selector joins")
}

// TestStakeCacheSelectorSwitch tests that both reporters recalculate stake when selector switches
// Scenario: Selector is with reporter A, switches to reporter B, both reporters submit
// Expected: Reporter A loses stake, Reporter B gains stake
func TestStakeCacheSelectorSwitch(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	config := stakeCacheSetupConfig()
	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// Turn on minting
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Node))

	// Both validators become reporters
	for i := range validators {
		minStakeAmt := "1000000"
		moniker := fmt.Sprintf("reporter_%d", i)
		txHash, err := validators[i].Node.ExecTx(ctx, validators[i].AccAddr, "reporter", "create-reporter", stakeCacheCommissRate, minStakeAmt, moniker, "--keyring-dir", validators[i].Node.HomeDir())
		require.NoError(err)
		fmt.Printf("TX HASH (validator %d becomes reporter): %s\n", i, txHash)
	}

	// Create a user with stake
	fundAmt := math.NewInt(2_000 * 1e6)
	delegateAmt := sdk.NewCoin("loya", math.NewInt(1000*1e6))
	user := interchaintest.GetAndFundTestUsers(t, ctx, "selector", fundAmt, chain)[0]

	// User delegates to validator 0
	txHash, err := validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[0].ValAddr, delegateAmt.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user delegates to val 0):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// User selects validator 0 as their reporter initially
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", validators[0].AccAddr, "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user selects validator 0 as reporter):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Both reporters submit first report
	currentCycleList := waitForStakeCacheCycleListQuery(t, ctx, validators[0].Node, "")
	firstQueryMetaID := currentCycleList.QueryMeta.Id

	value := layerutil.EncodeValue(300.0)
	for i := range validators {
		submitStakeCacheValue(t, ctx, validators[i].Node, currentCycleList.QueryData, value)
		fmt.Printf("Validator %d first report submitted\n", i)
	}

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Get individual reporter powers from micro-reports before switch
	var reporter0PowerBefore, reporter1PowerBefore uint64
	for i, v := range validators {
		reports, _, err := e2e.QueryWithTimeout(ctx, v.Node, "oracle", "get-reportsby-reporter", v.AccAddr, "--page-limit", "1")
		require.NoError(err)
		var reportsRes e2e.QueryMicroReportsResponse
		require.NoError(json.Unmarshal(reports, &reportsRes))
		require.NotEmpty(reportsRes.MicroReports, "Validator %d should have a micro-report", i)
		power, err := strconv.ParseUint(reportsRes.MicroReports[0].Power, 10, 64)
		require.NoError(err)
		if i == 0 {
			reporter0PowerBefore = power
		} else {
			reporter1PowerBefore = power
		}
		fmt.Printf("Validator %d power before switch: %d\n", i, power)
	}

	// Reporter 0 should have more power (includes selector's delegation)
	require.Greater(reporter0PowerBefore, reporter1PowerBefore, "Reporter 0 should have more power than reporter 1 before switch (has selector)")

	// User switches reporter from validator 0 to validator 1
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "switch-reporter", validators[1].AccAddr, "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user switches to validator 1):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Both reporters submit second report
	currentCycleList = waitForStakeCacheCycleListQuery(t, ctx, validators[0].Node, firstQueryMetaID)

	for i := range validators {
		submitStakeCacheValue(t, ctx, validators[i].Node, currentCycleList.QueryData, value)
		fmt.Printf("Validator %d second report submitted\n", i)
	}

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Get individual reporter powers after switch
	var reporter0PowerAfter, reporter1PowerAfter uint64
	for i, v := range validators {
		reports, _, err := e2e.QueryWithTimeout(ctx, v.Node, "oracle", "get-reportsby-reporter", v.AccAddr, "--page-limit", "2")
		require.NoError(err)
		var reportsRes e2e.QueryMicroReportsResponse
		require.NoError(json.Unmarshal(reports, &reportsRes))
		require.GreaterOrEqual(len(reportsRes.MicroReports), 2, "Validator %d should have at least 2 micro-reports", i)
		// Get the most recent report (last one)
		latestReport := reportsRes.MicroReports[len(reportsRes.MicroReports)-1]
		power, err := strconv.ParseUint(latestReport.Power, 10, 64)
		require.NoError(err)
		if i == 0 {
			reporter0PowerAfter = power
		} else {
			reporter1PowerAfter = power
		}
		fmt.Printf("Validator %d power after switch: %d\n", i, power)
	}

	// After switch: reporter 0 should lose selector's stake.
	// Reporter 1 does NOT gain the selector's stake yet because the selector is locked
	// for the unbonding period after switching reporters (LockedUntilTime is set in SwitchReporter).
	// GetReporterStake skips selectors whose LockedUntilTime is after the current block time.
	fmt.Printf("Reporter 0: %d -> %d\n", reporter0PowerBefore, reporter0PowerAfter)
	fmt.Printf("Reporter 1: %d -> %d\n", reporter1PowerBefore, reporter1PowerAfter)
	require.Less(reporter0PowerAfter, reporter0PowerBefore, "Reporter 0 should lose power after selector switches away")
	require.Equal(reporter1PowerAfter, reporter1PowerBefore, "Reporter 1 should not gain power yet (selector is locked for unbonding period)")
}

// TestStakeCacheDelegationChange tests that reporter stake is recalculated after delegation change
// Scenario: Selector delegates more to validator, reporter submits
// Expected: Reporter stake should reflect the new delegation amount
func TestStakeCacheDelegationChange(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	e2e.PrintValidatorInfo(ctx, validators)

	// Turn on minting
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Node))

	// Validator 0 becomes a reporter
	minStakeAmt := "1000000"
	txHash, err := validators[0].Node.ExecTx(ctx, validators[0].AccAddr, "reporter", "create-reporter", stakeCacheCommissRate, minStakeAmt, moniker, "--keyring-dir", validators[0].Node.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (validator 0 becomes reporter):", txHash)

	// Create a user with stake
	fundAmt := math.NewInt(10_000 * 1e6)
	initialDelegate := sdk.NewCoin("loya", math.NewInt(1000*1e6))
	user := interchaintest.GetAndFundTestUsers(t, ctx, "selector", fundAmt, chain)[0]

	// User delegates initial amount
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[0].ValAddr, initialDelegate.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user initial delegation):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// User selects validator 0 as their reporter
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", validators[0].AccAddr, "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user selects reporter):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// First report
	currentCycleListRes, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	require.NoError(json.Unmarshal(currentCycleListRes, &currentCycleList))

	value := layerutil.EncodeValue(400.0)
	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	fmt.Println("First report submitted")

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Query first report power
	qDataBz, err := hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz := utils.QueryIDFromData(qDataBz)
	qId := hex.EncodeToString(qIdBz)
	res, _, err := e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	var aggRes e2e.QueryGetCurrentAggregateReportResponse
	require.NoError(json.Unmarshal(res, &aggRes))
	firstPower, err := strconv.ParseUint(aggRes.Aggregate.AggregatePower, 10, 64)
	require.NoError(err)
	fmt.Println("First report power:", firstPower)

	// User delegates more (this triggers AfterDelegationModified hook)
	additionalDelegate := sdk.NewCoin("loya", math.NewInt(5000*1e6))
	txHash, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[0].ValAddr, additionalDelegate.String(), "--keyring-dir", validators[0].Node.HomeDir(), "--fees", stakeCacheTxFee)
	require.NoError(err)
	fmt.Println("TX HASH (user additional delegation):", txHash)

	require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Node))

	// Second report - should recalculate with new stake
	currentCycleListRes, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "current-cyclelist-query")
	require.NoError(err)
	require.NoError(json.Unmarshal(currentCycleListRes, &currentCycleList))

	submitStakeCacheValue(t, ctx, validators[0].Node, currentCycleList.QueryData, value)
	fmt.Println("Second report submitted")

	require.NoError(testutil.WaitForBlocks(ctx, 2, validators[0].Node))

	// Query second report power
	qDataBz, err = hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz = utils.QueryIDFromData(qDataBz)
	qId = hex.EncodeToString(qIdBz)
	res, _, err = e2e.QueryWithTimeout(ctx, validators[0].Node, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	require.NoError(json.Unmarshal(res, &aggRes))
	secondPower, err := strconv.ParseUint(aggRes.Aggregate.AggregatePower, 10, 64)
	require.NoError(err)
	fmt.Println("Second report power:", secondPower)

	// Second power should be greater due to additional delegation
	require.Greater(secondPower, firstPower, "Reporter stake should increase after additional delegation")
}
