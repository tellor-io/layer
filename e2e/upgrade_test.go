package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	util "github.com/tellor-io/layer/testutil"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	haltHeightDelta    = 12 // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = 12
)

func TestLayerUpgrade(t *testing.T) {
	ChainUpgradeTest(t, "layer", "layer", "local", "v5.1.1")
}

func ChainUpgradeTest(t *testing.T, chainName, upgradeContainerRepo, upgradeVersion, upgradeName string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
	}

	nv := 1
	nf := 0
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		{
			NumValidators: &nv,
			NumFullNodes:  &nf,
			ChainConfig: ibc.ChainConfig{
				Type:           "cosmos",
				Name:           "layer",
				ChainID:        "layer",
				Bin:            "layerd",
				Denom:          "loya",
				Bech32Prefix:   "tellor",
				CoinType:       "118",
				GasPrices:      "0.0loya",
				GasAdjustment:  1.1,
				TrustingPeriod: "504h",
				NoHostMount:    false,
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/tellor-io/layer",
						Version:    "v5.0.0",
						UidGid:     "1025:1025",
					},
				},
				EncodingConfig:      e2e.LayerEncoding(),
				ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
				AdditionalStartArgs: []string{"--key-name", "validator"},
			},
		},
	})

	client, network := interchaintest.DockerSetup(t)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	ctx := context.Background()

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	validatorI := chain.Validators[0]
	valAddr, err := validatorI.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)

	// create reporter, submit 1 normal report, submit 1 no stake report
	_, err = validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val1_moniker", "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)
	// tip
	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", chain.HomeDir()), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)
	// submit-value
	_, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)
	// submit-value
	_, err = validatorI.ExecTx(ctx, "validator", "oracle", "no-stake-report", qData, value, "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)

	// create user to send upgrade tx
	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, chain)
	chainUser := users[0]

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta

	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     "500000000" + chain.Config().Denom, // greater than min deposit
		Title:       "Chain Upgrade 1",
		Name:        upgradeName,
		Description: "First chain software upgrade",
		Height:      haltHeight,
	}

	// submit upgrade proposal
	upgradeTx, err := chain.UpgradeProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err, "error submitting software upgrade proposal tx")

	propId, err := strconv.ParseUint(upgradeTx.ProposalID, 10, 64)
	require.NoError(t, err, "failed to convert proposal ID to uint64")

	// vote on proposal
	err = chain.VoteOnProposalAllValidators(ctx, propId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	// get proposal status
	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, propId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should timeout due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, "height is not equal to halt height")

	// bring down nodes to prepare for upgrade
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	chain.UpgradeVersion(ctx, client, upgradeContainerRepo, upgradeVersion)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = chain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*240)
	defer timeoutCtxCancel()

	// Enhanced testing after upgrade: 10 more tips and reports with different scenarios
	fmt.Println("=== Testing post-upgrade oracle functionality ===")

	// Test 1: Submit 10 more tip+report cycles with different values
	for i := 0; i < 10; i++ {
		// Generate different values for each report
		testValue := util.EncodeValue(float64(1000000 + i*100000))

		// tip
		_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", chain.HomeDir()), validatorI.Chain.Config().Env)
		require.NoError(t, err, fmt.Sprintf("error tipping for report %d", i+1))

		err = testutil.WaitForBlocks(ctx, 1, validatorI)
		require.NoError(t, err)

		// submit-value
		_, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", qData, testValue, "--keyring-dir", chain.HomeDir())
		require.NoError(t, err, fmt.Sprintf("error submitting value for report %d", i+1))

		err = testutil.WaitForBlocks(ctx, 1, validatorI)
		require.NoError(t, err)

		fmt.Printf("Completed tip+report cycle %d with value %s\n", i+1, testValue)
	}

	// Test 2: Submit 5 more no-stake reports with different values
	for i := 0; i < 5; i++ {
		testValue := util.EncodeValue(float64(2000000 + i*50000))
		_, err = validatorI.ExecTx(ctx, "validator", "oracle", "no-stake-report", qData, testValue, "--keyring-dir", chain.HomeDir())
		require.NoError(t, err, fmt.Sprintf("error submitting no-stake report %d", i+1))

		err = testutil.WaitForBlocks(ctx, 1, validatorI)
		require.NoError(t, err)

		fmt.Printf("Completed no-stake report %d with value %s\n", i+1, testValue)
	}

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// Helper function for query with timeout
	queryWithTimeout := func(args ...string) ([]byte, []byte, error) {
		queryCtx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		return validatorI.ExecQuery(queryCtx, args...)
	}

	// Test 3: Query reports with different pagination scenarios
	// Should now have 10 regular reports (10 new)
	reports, _, err := queryWithTimeout("oracle", "get-reportsby-reporter", valAddr)
	require.NoError(t, err, "error querying reports by reporter")
	var reportsRes e2e.QueryMicroReportsResponse
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling reports response")
	fmt.Printf("Total reports found: %d\n", len(reportsRes.MicroReports))
	require.Equal(t, valAddr, reportsRes.MicroReports[0].Reporter)
	require.Equal(t, 10, len(reportsRes.MicroReports), "Should have 11 regular reports after upgrade")

	// Test 4: Query no-stake reports - should have 5 new
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr)
	require.NoError(t, err, "error querying no-stake reports")
	var reportsResNoStake e2e.QueryGetReportersNoStakeReportsResponse
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling no-stake reports response")
	fmt.Printf("Total no-stake reports found: %d\n", len(reportsResNoStake.NoStakeReports))
	require.Equal(t, valAddr, reportsResNoStake.NoStakeReports[0].Reporter)
	require.Equal(t, 5, len(reportsResNoStake.NoStakeReports), "Should have 5 no-stake reports after upgrade")

	// Test 5: Pagination edge cases with limit
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-limit", "5")
	require.NoError(t, err, "error querying reports with page limit")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling paginated reports")
	require.Equal(t, 5, len(reportsRes.MicroReports), "Should respect page limit of 5")

	// Test 6: Pagination with large limit
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-limit", "20")
	require.NoError(t, err, "error querying reports with large limit")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling reports with large limit")
	require.Equal(t, 10, len(reportsRes.MicroReports), "Should return all 11 reports when limit is higher")

	// Test 7: Pagination with offset
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-offset", "5", "--page-limit", "3")
	require.NoError(t, err, "error querying reports with offset")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling reports with offset")
	require.Equal(t, 3, len(reportsRes.MicroReports), "Should return 3 reports with offset 5")

	// Test 8: Reverse pagination
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-reverse", "--page-limit", "3")
	require.NoError(t, err, "error querying reports in reverse")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling reversed reports")
	require.Equal(t, 3, len(reportsRes.MicroReports), "Should return 3 reports in reverse order")

	// Test 9: No-stake reports pagination scenarios
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr, "--page-limit", "3")
	require.NoError(t, err, "error querying no-stake reports with limit")
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling no-stake reports with limit")
	require.Equal(t, 3, len(reportsResNoStake.NoStakeReports), "Should return 3 no-stake reports with limit")

	// Test 10: No-stake reports with offset and reverse
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr, "--page-offset", "2", "--page-reverse", "--page-limit", "2")
	require.NoError(t, err, "error querying no-stake reports with offset and reverse")
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling no-stake reports with offset and reverse")
	require.Equal(t, 2, len(reportsResNoStake.NoStakeReports), "Should return 2 no-stake reports with offset and reverse")

	// Test 11: Zero limit edge case, should default to 10
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-limit", "0")
	require.NoError(t, err, "error querying reports with zero limit")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling reports with zero limit")
	require.Equal(t, 10, len(reportsRes.MicroReports), "Should return 0 reports when limit is 0")

	// Additional compatibility tests
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-limit", "10", "--page-reverse", "--page-offset", "1")
	require.NoError(t, err, "error in compatibility test for reports")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling compatibility test reports")
	require.Equal(t, 9, len(reportsRes.MicroReports), "Compatibility test should return 10 reports")

	// query no stake reports with all flags
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr, "--page-limit", "10", "--page-reverse", "--page-offset", "1")
	require.NoError(t, err, "error in compatibility test for no-stake reports")
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling compatibility test no-stake reports")
	require.Equal(t, 4, len(reportsResNoStake.NoStakeReports), "Compatibility test should return 5 no-stake reports")

	// Test individual flags for get-reportsby-reporter
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-limit", "5")
	require.NoError(t, err, "error testing individual page-limit flag")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling individual flag test")
	require.Equal(t, 5, len(reportsRes.MicroReports), "Individual flag test should return 5 reports")

	// query reports by reporter with only page-reverse flag
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-reverse")
	require.NoError(t, err, "error testing page-reverse flag")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling page-reverse test")
	require.Equal(t, 10, len(reportsRes.MicroReports), "Page-reverse test should return all 10 reports")

	// query reports by reporter with only page-offset flag
	reports, _, err = queryWithTimeout("oracle", "get-reportsby-reporter", valAddr, "--page-offset", "0")
	require.NoError(t, err, "error testing page-offset flag")
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err, "error unmarshaling page-offset test")
	require.Equal(t, 10, len(reportsRes.MicroReports), "Page-offset test should return all 10 reports")

	// Test individual flags for get-reporters-no-stake-reports
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr, "--page-reverse")
	require.NoError(t, err, "error testing no-stake reports page-reverse")
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling no-stake reports page-reverse test")
	require.Equal(t, 5, len(reportsResNoStake.NoStakeReports), "No-stake reports page-reverse should return 5 reports")

	// query no stake reports by reporter with only page-offset flag
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr, "--page-offset", "0")
	require.NoError(t, err, "error testing no-stake reports page-offset")
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling no-stake reports page-offset test")
	require.Equal(t, 5, len(reportsResNoStake.NoStakeReports), "No-stake reports page-offset should return 5 reports")

	// Final simple verification
	reports, _, err = queryWithTimeout("oracle", "get-reporters-no-stake-reports", valAddr, "--page-limit", "1")
	require.NoError(t, err, "error in final verification")
	err = json.Unmarshal(reports, &reportsResNoStake)
	require.NoError(t, err, "error unmarshaling final verification")
	require.Equal(t, valAddr, reportsResNoStake.NoStakeReports[0].Reporter)
	require.Equal(t, 1, len(reportsResNoStake.NoStakeReports))

	fmt.Println("=== All tests completed successfully! ===")
}
