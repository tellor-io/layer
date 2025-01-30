package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestDispute --timeout 5m

// open 10 disputes at the same time
func TestDispute(t *testing.T) {
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
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
	}

	nv := 2
	nf := 2
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
				AdditionalStartArgs: []string{"--key-name", "validator", "--price-daemon-enabled=false"},
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
	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)
	val2 := chain.Validators[1]
	val2Addr, err := val2.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val2valAddr, err := val2.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val2 Account Address: ", val2Addr)
	fmt.Println("val2 Validator Address: ", val2valAddr)

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// get val1 staking power
	val1Staking, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	val1StartPower := val1Staking.Tokens
	fmt.Println("val1 staking power before delegations: ", val1StartPower)

	type ReporterAccs struct {
		Keyname string
		Addr    string
	}

	// make 10 users who will delegate to val2 and become reporters
	reporters := make([]ReporterAccs, 10)
	expectedDelTotal := math.NewInt(0)
	for i := 0; i < 10; i++ {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt := sdk.NewCoin("loya", math.NewInt(1_000*1e6))
		user := interchaintest.GetAndFundTestUsers(t, ctx, keyname, fundAmt, chain)[0]
		txHash, err := val1.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", val1valAddr, delegateAmt.String(), "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (", keyname, " delegates to val1): ", txHash)
		reporters[i] = ReporterAccs{
			Keyname: keyname,
			Addr:    user.FormattedAddress(),
		}
		expectedDelTotal = expectedDelTotal.Add(delegateAmt.Amount)
		fmt.Println("expectedDelTotal: ", expectedDelTotal)
		val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
		require.NoError(err)
		fmt.Println("val1 staking power: ", val1Staking.Tokens)
	}
	fmt.Println("reporters: ", reporters)
	fmt.Println("expectedDelTotal: ", expectedDelTotal)

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1StartPower.Add(expectedDelTotal))

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, val1))

	// custom gov params set voting period to 15s
	require.NoError(testutil.WaitForBlocks(ctx, 5, val1))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)

	expectedYesTotal := math.NewInt(10000000000000).Add(expectedDelTotal)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.FinalTallyResult.Yes.String(), expectedYesTotal.String())
	require.Equal(result.FinalTallyResult.No.String(), "0")
	require.Equal(result.FinalTallyResult.Abstain.String(), "0")
	require.Equal(result.FinalTallyResult.NoWithVeto.String(), "0")
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// all 10 delegators become reporters
	for i := 0; i < len(reporters); i++ {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, reportersRes)
	require.NoError(err)
	fmt.Println("reporters ress: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), 10)

	// // validatorI becomes a reporter
	// txHash, err := validatorI.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", "/var/cosmos-chain/layer-1")
	// require.NoError(err)
	// fmt.Println("TX HASH (validatorI becomes a reporter): ", txHash)

	// // user tips random amount (<1 trb + 1 loya) for LTC/USD
	// randomTipInt := rand.Int63n(1000000) + 1
	// randomTip := sdk.NewCoin("loya", math.NewInt(randomTipInt))
	// fmt.Println("ltc/usd tip: ", randomTip.String())

	// stdout, _, err := validatorI.Exec(ctx, validatorI.TxCommand("user1", "oracle", "tip", user1Addr, ltcusdQData, randomTip.String(), "--keyring-dir", "/var/cosmos-chain/layer-1"), validatorI.Chain.Config().Env)
	// require.NoError(err)
	// txHash, err = e2e.GetTxHashFromExec(stdout)
	// fmt.Println("TX HASH (user tips ltc/usd): ", txHash)
	// require.NoError(testutil.WaitForBlocks(ctx, 1, validatorI))

	// // validator/reporter submits good value for LTC/USD
	// ltcusdValue := layerutil.EncodeValue(75.98)
	// valI, err := layer.StakingQueryValidator(ctx, valAddress)
	// require.NoError(err)
	// expectedPower := valI.Tokens // loya

	// txHash, err = validatorI.ExecTx(ctx, "validator", "oracle", "submit-value", valAccAddress, ltcusdQData, ltcusdValue, "--keyring-dir", "/var/cosmos-chain/layer-1")
	// require.NoError(err)
	// fmt.Println("TX HASH (user reports LTC/USD): ", txHash)
	// // require.NoError(testutil.WaitForBlocks(ctx, 1, validatorI))

	// // make sure all is square on aggregate report
	// ltcusdReport, _, err := validatorI.ExecQuery(ctx, "oracle", "get-reportsby-reporter", valAccAddress)
	// require.NoError(err)
	// var microReports e2e.ReportsResponse
	// require.NoError(json.Unmarshal(ltcusdReport, &microReports))

	// require.Equal(microReports.MicroReports[0].Reporter, valAccAddress)
	// require.Equal(microReports.MicroReports[0].Value, ltcusdValue)
	// require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
	// require.Equal(microReports.MicroReports[0].Power, expectedPower.QuoRaw(layertypes.PowerReduction.Int64()).String()) // power is in trb
	// require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")
	// // require.Equal(microReports.MicroReports[0].QueryID, ltcusdQId) // GVhdkSr7cjeOOYanpT8erh+655LNF+HQ3wY2gTJoI64= expected ?
	// txResp, err := validatorI.TxHashToResponse(ctx, txHash)
	// fmt.Println("txResp.Events: ", txResp.Events[len(txResp.Events)-1])
	// fmt.Println("txResp.Logs: ", txResp.Logs)
	// fmt.Println("txResp.Tx: ", txResp.Tx)
	// fmt.Println("txResp.Height: ", txResp.Height)
	// // txQuery, _, err := validatorI.ExecQuery(ctx, "tx", txHash)
	// // require.NotNil(txQuery)
	// // require.NoError(err)
	// // fmt.Println("tx query: ", string(txQuery))
	// // require.Equal(microReports.MicroReports[0].BlockNumber, blockNum)
	// // require.Equal(microReports.MicroReports[0].Timestamp, timestamp)

	// // user opens warning dispute on report
	// bz, err := json.Marshal(microReports.MicroReports[0])
	// require.NoError(err)

	// txHash, err = validatorI.ExecTx(ctx, user1Addr, "dispute", "propose-dispute", string(bz), "warning", "500000000000loya", "false", "--keyring-dir", "/var/cosmos-chain/layer-1", "--gas", "1000000", "--fees", "1000000loya")
	// require.NoError(err)
	// fmt.Println("TX HASH (user opens warning dispute on report): ", txHash)

}
