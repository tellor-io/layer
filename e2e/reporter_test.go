package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestSelectorCreateReporter(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "20s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
	}

	nv := 2
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

	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

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

	// create user selector
	fundAmt := math.NewInt(1_100 * 1e6)
	delegateAmt := sdk.NewCoin("loya", math.NewInt(1000*1e6)) // all tokens after paying fee
	user := interchaintest.GetAndFundTestUsers(t, ctx, "user", fundAmt, chain)[0]
	txHash, err := validators[0].Val.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", validators[1].ValAddr, delegateAmt.String(), "--keyring-dir", validators[0].Val.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
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
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Val))
	require.NoError(testutil.WaitForBlocks(ctx, 7, validators[0].Val))
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

	// user selects val 1 as their reporter
	txHash, err = validators[0].Val.ExecTx(ctx, user.FormattedAddress(), "reporter", "select-reporter", validators[1].Addr, "--keyring-dir", validators[0].Val.HomeDir(), "--gas", "1000000", "--fees", "10loya")
	require.NoError(err)
	fmt.Println("TX HASH (user selects val1 as their reporter): ", txHash)

	//  both reporters submit for cyclelist
	currentCycleListRes, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Println("current cycle list: ", currentCycleList)
	value := layerutil.EncodeValue(123456789.99)
	for i := range validators {
		_, _, err = validators[i].Val.Exec(ctx, validators[i].Val.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", validators[i].Val.HomeDir()), validators[i].Val.Chain.Config().Env)
		require.NoError(err)
		height, err := validators[i].Val.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}

	// wait for aggregation to complete
	require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Val))

	// query report info
	qDataBz, err := hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz := utils.QueryIDFromData(qDataBz)
	qId := hex.EncodeToString(qIdBz)
	res, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	fmt.Println("current aggregate report power: ", currentAggRes.Aggregate.AggregatePower)
	report1Power := currentAggRes.Aggregate.AggregatePower

	// user creates a reporter
	minStakeAmt := "1000000"
	moniker := "reporter_moniker"
	txHash, err = validators[0].Val.ExecTx(ctx, user.FormattedAddress(), "reporter", "create-reporter", commissRate, minStakeAmt, moniker, "--keyring-dir", validators[0].Val.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user creates a reporter): ", txHash)

	// all 3 reporters report
	currentCycleListRes, _, err = validators[0].Val.ExecQuery(ctx, "oracle", "current-cyclelist-query")
	require.NoError(err)
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	for i := range validators {
		_, _, err = validators[i].Val.Exec(ctx, validators[i].Val.TxCommand("validator", "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", validators[i].Val.HomeDir()), validators[i].Val.Chain.Config().Env)
		require.NoError(err)
		height, err := validators[i].Val.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}
	_, _, err = validators[0].Val.Exec(ctx, validators[0].Val.TxCommand(user.FormattedAddress(), "oracle", "submit-value", currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", validators[0].Val.HomeDir()), validators[0].Val.Chain.Config().Env)
	require.NoError(err)

	// wait for aggregation to complete
	require.NoError(testutil.WaitForBlocks(ctx, 3, validators[0].Val))

	// query report info
	qDataBz, err = hex.DecodeString(currentCycleList.QueryData)
	require.NoError(err)
	qIdBz = utils.QueryIDFromData(qDataBz)
	qId = hex.EncodeToString(qIdBz)
	res, _, err = validators[0].Val.ExecQuery(ctx, "oracle", "get-current-aggregate-report", qId)
	require.NoError(err)
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	fmt.Println("current aggregate report power: ", currentAggRes.Aggregate.AggregatePower)
	report2Power := currentAggRes.Aggregate.AggregatePower
	require.Less(report2Power, report1Power) // report 1 should have more than 2
}
