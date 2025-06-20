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

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	haltHeightDelta    = 12 // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = 12
)

func TestLayerUpgrade(t *testing.T) {
	// t.Skip("needs to switch between binaries to run successfully")
	ChainUpgradeTest(t, "layer", "layerup", "local", "v5.1.0")
}

func ChainUpgradeTest(t *testing.T, chainName, upgradeContainerRepo, upgradeVersion, upgradeName string) {
	t.Helper()
	if testing.Short() {
		// t.Skip("skipping in short mode")
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
						Repository: "layer",
						Version:    "local",
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

	_, err = validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val1_moniker", "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)

	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", chain.HomeDir()), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)

	// value submitted on old version
	_, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)

	// also submit a no stake report
	_, err = validatorI.ExecTx(ctx, "validator", "oracle", "no-stake-report", qData, value, "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)

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

	upgradeTx, err := chain.UpgradeProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err, "error submitting software upgrade proposal tx")

	propId, err := strconv.ParseUint(upgradeTx.ProposalID, 10, 64)
	require.NoError(t, err, "failed to convert proposal ID to uint64")

	err = chain.VoteOnProposalAllValidators(ctx, propId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

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

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()
	_, _, err = validatorI.Exec(ctx, validatorI.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", chain.HomeDir()), validatorI.Chain.Config().Env)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, validatorI)
	require.NoError(t, err)

	_, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)
	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// submit another no stake report
	_, err = validatorI.ExecTx(ctx, "validator", "oracle", "no-stake-report", qData, value, "--keyring-dir", chain.HomeDir())
	require.NoError(t, err)

	// query old report by reporter
	reports, _, err := validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAddr, "--page-limit", "1")
	require.NoError(t, err)
	// unmarshal
	var reportsRes e2e.QueryMicroReportsResponse
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err)
	fmt.Println("length: ", len(reportsRes.MicroReports))
	fmt.Println("reports: ", reportsRes)
	require.Equal(t, valAddr, reportsRes.MicroReports[0].Reporter)
	blockNum, err := strconv.ParseInt(reportsRes.MicroReports[0].BlockNumber, 10, 64)
	require.NoError(t, err)
	require.Less(t, blockNum, haltHeight)

	// query all reports by reporter, should be 2
	reports, _, err = validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAddr)
	require.NoError(t, err)
	// unmarshal
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err)
	require.Equal(t, 2, len(reportsRes.MicroReports))

	// query old no stake reports by reporter
	reports, _, err = validatorI.ExecQuery(ctx, "oracle", "get-reporters-no-stake-reports", valAddr, "--page-limit", "1")
	require.NoError(t, err)
	// unmarshal
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err)
	fmt.Println("length: ", len(reportsRes.MicroReports))
	fmt.Println("reports: ", reportsRes)
	require.Equal(t, valAddr, reportsRes.MicroReports[0].Reporter)
	blockNum, err = strconv.ParseInt(reportsRes.MicroReports[0].BlockNumber, 10, 64)
	require.NoError(t, err)
	require.Less(t, blockNum, haltHeight)

	// query all no stake reports by reporter, should be 2
	reports, _, err = validatorI.ExecQuery(ctx, "oracle", "get-reporters-no-stake-reports", valAddr)
	require.NoError(t, err)
	// unmarshal
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err)
	require.Equal(t, 2, len(reportsRes.MicroReports))

	// query new report by reporter
	reports, _, err = validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAddr, "--page-reverse", "--page-limit", "1")
	require.NoError(t, err)
	// unmarshal
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err)
	fmt.Println("length: ", len(reportsRes.MicroReports))
	fmt.Println("reports: ", reportsRes)
	require.Equal(t, valAddr, reportsRes.MicroReports[0].Reporter)
	blockNum, err = strconv.ParseInt(reportsRes.MicroReports[0].BlockNumber, 10, 64)
	require.NoError(t, err)
	require.Greater(t, blockNum, haltHeight)

	// query new no stake reports by reporter
	reports, _, err = validatorI.ExecQuery(ctx, "oracle", "get-reporters-no-stake-reports", valAddr, "--page-reverse")
	require.NoError(t, err)
	// unmarshal
	err = json.Unmarshal(reports, &reportsRes)
	require.NoError(t, err)
	fmt.Println("length: ", len(reportsRes.MicroReports))
	fmt.Println("reports: ", reportsRes)
	require.Equal(t, valAddr, reportsRes.MicroReports[0].Reporter)
	blockNum, err = strconv.ParseInt(reportsRes.MicroReports[0].BlockNumber, 10, 64)
	require.NoError(t, err)
	require.Greater(t, blockNum, haltHeight)
}
