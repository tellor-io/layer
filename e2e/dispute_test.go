package e2e_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
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
	layerutil "github.com/tellor-io/layer/testutil"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	trxQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747278000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	trxQId    = "954476140bd7309c72b6bdc8d71a293ec3df5ad00b79809dc21c98f7fc495bfb"
	suiQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003737569000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	suiQId    = "8f76558fd2800ccaeb236d250830d068a8d9fb0568fe1b32fc916386558547f4"
	bchQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003626368000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	bchQId    = "efa84ae5ea9eb0545e159f78f0a44911ac5a81ecb6ff0c4e32107bcfc66c4baa"
	ltcQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000036c7463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	ltcQId    = "19585d912afb72378e3986a7a53f1eae1fbae792cd17e1d0df063681326823ae"
	solQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003736f6c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	solQId    = "b211d6f1abbd5bb431618547402a92250b765151acbe749e7f9c26dc19e5dd9a"
	dogeQData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004646f67650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	dogeQId   = "15d3cb16e8175919781af07b2ce06714d24f168284b1b47b14b6bfbe9a5a02ff"
	dotQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003646f74000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	dotQId    = "8810ffb0cfcb6131da29ed4b229f252d6bac6fc98fc4a61ffbde5b48131e0228"
	bnbQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003626e62000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	bnbQId    = "235b53e7caaba06517ae5af902e0e765b4032e8a75b82fd832c4da22486e47b4"
	xrpQData  = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003787270000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	xrpQId    = "ba615496c4671e5b931b0bbd81046d3f63fb453c414a830d6c4f923864eebf8b"
	hypeQData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004687970650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	hypeQId   = "48318e44abe415e4eabf291f1aab42a0af0e87ca25868e86b07df7f385b2ff81"
)

type QueryData struct {
	QueryData string
	QueryID   string
}

type ReporterAccs struct {
	Keyname string
	Addr    string
}

// cd e2e
// go test -run TestDispute --timeout 5m

// open 10 disputes simultaneously, vote and resolve all of them
// 10 disputes on 10 different ppl
func TestTenDisputesTenPeople(t *testing.T) {
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
	nf := 1
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

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
	val2Staking, err := chain.StakingQueryValidator(ctx, val2valAddr)
	require.NoError(err)
	val2StartPower := val2Staking.Tokens
	fmt.Println("val2 staking power before delegations: ", val2StartPower)

	// make 10 users who will delegate to val2 and become reporters
	numReporters := 10
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	for i := 0; i < numReporters; i++ {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(100_000 * 1e6)
		delegateAmt := sdk.NewCoin("loya", math.NewInt(1_000*1e6))
		user := interchaintest.GetAndFundTestUsers(t, ctx, keyname, fundAmt, chain)[0]
		txHash, err := val1.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", val2valAddr, delegateAmt.String(), "--keyring-dir", val2.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (", keyname, " delegates to val2): ", txHash)
		reporters[i] = ReporterAccs{
			Keyname: keyname,
			Addr:    user.FormattedAddress(),
		}
		expectedDelTotal = expectedDelTotal.Add(delegateAmt.Amount)
		fmt.Println("expectedDelTotal: ", expectedDelTotal)
		val2Staking, err = chain.StakingQueryValidator(ctx, val2valAddr)
		require.NoError(err)
		fmt.Println("val2 staking power: ", val2Staking.Tokens)
	}
	fmt.Println("reporters: ", reporters)
	fmt.Println("expectedDelTotal: ", expectedDelTotal)

	// get val2 staking power
	val2Staking, err = chain.StakingQueryValidator(ctx, val2valAddr)
	require.NoError(err)
	fmt.Println("val2 staking power: ", val2Staking.Tokens)
	require.Equal(val2Staking.Tokens, val2StartPower.Add(expectedDelTotal))

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
	// since reporters delegated 1000 trb each to val2, they will have 1000 reporting power
	for i := 0; i < len(reporters); i++ {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// val1 becomes a reporter
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val1 becomes a reporter): ", txHash)

	// val2 becomes a reporter
	txHash, err = val2.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val2.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val2 becomes a reporter): ", txHash)

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+2) // number of delegating reporters + 2 validator reporters

	// tip 1trb and report for 10 different spotprices
	// needs to be the same length as numReporters
	queryDataList := []QueryData{
		{QueryData: bchQData, QueryID: bchQId},
		{QueryData: ltcQData, QueryID: ltcQId},
		{QueryData: solQData, QueryID: solQId},
		{QueryData: dogeQData, QueryID: dogeQId},
		{QueryData: dotQData, QueryID: dotQId},
		{QueryData: bnbQData, QueryID: bnbQId},
		{QueryData: xrpQData, QueryID: xrpQId},
		{QueryData: hypeQData, QueryID: hypeQId},
		{QueryData: trxQData, QueryID: trxQId},
		{QueryData: suiQData, QueryID: suiQId},
	}
	value := layerutil.EncodeValue(10000000.99)
	tipAmt := math.NewInt(1 * 1e6)
	tip := sdk.NewCoin("loya", tipAmt)
	for i, query := range queryDataList {
		// tip 1 trb
		_, _, err := val1.Exec(ctx, val1.TxCommand("validator", "oracle", "tip", val1Addr, query.QueryData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
		require.NoError(err)
		fmt.Println("val1 tipped ", query.QueryID)
		err = testutil.WaitForBlocks(ctx, 1, val1)
		require.NoError(err)

		// report with 1000 reporting power
		txHash, err := val1.ExecTx(ctx, "validator", "oracle", "submit-value", reporters[i].Addr, query.QueryData, value, "--keyring-dir", val1.HomeDir())
		fmt.Println("TX HASH (", reporters[i].Keyname, " reported): ", txHash)
		require.NoError(err)

		// wait for query to expire and dispute
		err = testutil.WaitForBlocks(ctx, 2, val1)
		require.NoError(err)
		microreport, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", reporters[i].Addr)
		require.NoError(err)
		var microReports e2e.ReportsResponse
		require.NoError(json.Unmarshal(microreport, &microReports))
		require.Equal(microReports.MicroReports[0].Reporter, reporters[i].Addr)
		require.Equal(microReports.MicroReports[0].Value, value)
		require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
		require.Equal(microReports.MicroReports[0].Power, "1000")
		require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")

		decodedBytes, err := base64.StdEncoding.DecodeString(microReports.MicroReports[0].QueryID)
		require.NoError(err)
		queryId := hex.EncodeToString(decodedBytes)

		// get disputer staking power before dispute
		disputerStakingBefore, err := chain.StakingQueryValidator(ctx, val1valAddr)
		require.NoError(err)
		fmt.Println("disputer staking power before dispute: ", disputerStakingBefore.Tokens)

		// get val2 staking power before dispute
		val2StakingBefore, err := chain.StakingQueryValidator(ctx, val2valAddr)
		require.NoError(err)
		fmt.Println("val2 staking power before dispute: ", val2StakingBefore.Tokens)

		// dispute from validator
		// since reporting power is 1000, first rd fee fee is 10 trb
		// paying from bond, so val1 stake should decrease by 10 trb
		// val2 stake should also decrease by 10 trb bc of slash on reporter delgated to them
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "propose-dispute", microReports.MicroReports[0].Reporter, microReports.MicroReports[0].MetaId, queryId, "warning", "500000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (dispute on ", microReports.MicroReports[0].Reporter, "): ", txHash)

		// check disputer staking power after dispute
		// should decrease by 10 trb for every dispute opened for paying fee
		disputerStakingAfter, err := chain.StakingQueryValidator(ctx, val1valAddr)
		require.NoError(err)
		fmt.Println("disputer staking power after dispute: ", disputerStakingAfter.Tokens)
		require.Equal(disputerStakingAfter.Tokens, disputerStakingBefore.Tokens.Sub(math.NewInt(10*1e6))) // expected fee is 10 trb because reporting power is 1000

		// check val2 staking power after dispute
		// should decrease by 10 trb for every dispute opened for reporter getting slashed
		val2StakingAfter, err := chain.StakingQueryValidator(ctx, val2valAddr)
		require.NoError(err)
		fmt.Println("val2 staking power after dispute: ", val2StakingAfter.Tokens)
		require.Equal(val2StakingAfter.Tokens, val2StakingBefore.Tokens.Sub(math.NewInt(10*1e6))) // expected fee is 10 trb becyuase reporting power is 1000
	}

	// check open disputes
	res, _, err = val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(res, &openDisputes))
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes)
	// require.Equal(len(openDisputes.OpenDisputes.Ids), 10) // all 10 disputes should be open

	// vote and resolve all disputes
	for i := 0; i < len(queryDataList); i++ {
		disputeId := strconv.Itoa(i + 1)
		// vote from val1 (all tipping power)
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val1 votes on dispute ", i+1, "): ", txHash)

		// vote from val2 (0 power error)
		_, err = val2.ExecTx(ctx, "validator", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val2.HomeDir())
		require.Error(err)

		// check dispute status
		// should still be open bc only 33% of power has voted
		res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		var disputes e2e.Disputes
		require.NoError(json.Unmarshal(res, &disputes))
		require.Equal(disputes.Disputes[i].Metadata.DisputeStatus, 1) // not resolved yet

		// vote from team (should be at least 66% voting power after (33% from team, 33% from having one tip from val1))
		txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (team votes on dispute ", disputeId, "): ", txHash)

		// check on dispute status
		// should be resolved and executed
		r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		err = json.Unmarshal(r, &disputes)
		require.NoError(err)
		require.Equal(disputes.Disputes[i].Metadata.DisputeStatus, 2) // resolved now
		fmt.Println("resolved dispute: ", disputes.Disputes[i].DisputeID)

		// check dispute feepayer balance before fee refund
		disputerStakeBeforeFeeClaim, err := chain.StakingQueryValidator(ctx, val1valAddr)
		require.NoError(err)
		fmt.Println("disputer stake before fee claim: ", disputerStakeBeforeFeeClaim.Tokens)
		// check other val staked tokens before fee refund - should not change
		val2StakedBeforeFeeClaim, err := chain.StakingQueryValidator(ctx, val2valAddr)
		require.NoError(err)
		fmt.Println("val2 staked tokens before fee claim: ", val2StakedBeforeFeeClaim.Tokens)
		// withdraw fee refund from disputer (fee paid to start dispute, and 1% of naughty reporters' stake since vote settled to support)
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "withdraw-fee-refund", val1Addr, disputeId, "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
		require.NoError(err)
		fmt.Println("TX HASH (disputer claims fee refund on dispute ", disputeId, "): ", txHash)
		// check feepayer balance after fee refund
		disputerStakeAfterFeeClaim, err := chain.StakingQueryValidator(ctx, val1valAddr)
		require.NoError(err)
		fmt.Println("disputer stake after fee claim: ", disputerStakeAfterFeeClaim.Tokens)
		expectedDisputeFeeRefund := math.NewInt(95 * 1e5)
		expectedReporterBondToFeePayers := math.NewInt(10 * 1e6)
		// total fee is 10 trb (10*1e6 loya), claim is 95% of that so 9.5 trb (95 * 1e5 loya)
		// reporter bond to fee payers is 10 trb (10*1e6 loya)
		require.Equal(disputerStakeAfterFeeClaim.Tokens, disputerStakeBeforeFeeClaim.Tokens.Add(expectedDisputeFeeRefund).Add(expectedReporterBondToFeePayers))
		// check other val staked tokens after fee refund
		// other val should not get any rewards
		val2StakedAfterFeeClaim, err := chain.StakingQueryValidator(ctx, val2valAddr)
		require.NoError(err)
		fmt.Println("val2 staked tokens after fee claim: ", val2StakedAfterFeeClaim.Tokens)
		require.Equal(val2StakedAfterFeeClaim.Tokens, val2StakedBeforeFeeClaim.Tokens)

		// claim reward from disputer (voting reward)
		disputerBalBeforeRewardClaim, err := chain.BankQueryBalance(ctx, val1Addr, "loya")
		require.NoError(err)
		fmt.Println("disputer balance before reward claim: ", disputerBalBeforeRewardClaim)
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "claim-reward", disputeId, "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
		require.NoError(err)
		fmt.Println("TX HASH (disputer claims reward on dispute ", disputeId, "): ", txHash)
		// check disputer balance after reward claim
		// disputer should get 100% of the voting reward, team gets 0 and val with all tipping power was only other person to vote
		disputerBalAfterRewardClaim, err := chain.BankQueryBalance(ctx, val1Addr, "loya")
		require.NoError(err)
		expectedVoterReward := math.NewInt(250000)
		ninetyNinePercentOfVotingReward := expectedVoterReward.Mul(math.NewInt(99)).Quo(math.NewInt(100))
		// make sure reward is less than 100% but greater than 99%
		require.Greater(disputerBalAfterRewardClaim.String(), disputerBalBeforeRewardClaim.Add(ninetyNinePercentOfVotingReward).String())
		require.Less(disputerBalAfterRewardClaim.String(), disputerBalBeforeRewardClaim.Add(expectedVoterReward).String())
		fmt.Println("disputer balance after reward claim: ", disputerBalAfterRewardClaim)

		// try to claim reward again - should fail
		_, err = val1.ExecTx(ctx, "validator", "dispute", "claim-reward", disputeId, "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
		require.Error(err)
		// try to claim fee refund again - should fail
		_, err = val1.ExecTx(ctx, "validator", "dispute", "withdraw-fee-refund", val1Addr, disputeId, "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
		require.Error(err)
	}
}

// reporter reports a bad value, unbonds some tokens, gets major disputed
func TestReportUnbondMajorDispute(t *testing.T) {
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// get val1 staking power
	val1Staking, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	val1StartPower := val1Staking.Tokens
	fmt.Println("val1 staking power before delegations: ", val1StartPower)

	// make 2 users who will delegate to val1 and become reporters
	numReporters := 2
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	var user0Addr string
	var user1Addr string
	for i := range numReporters {
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
		if i == 0 {
			user0Addr = user.FormattedAddress()
		} else {
			user1Addr = user.FormattedAddress()
		}
	}

	// query staking module delegations
	delegations, err := chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, user1

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1StartPower.Add(expectedDelTotal))
	val1power := val1Staking.Tokens

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

	// all 2 delegators become reporters
	for i := range reporters {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// val1 becomes a reporter
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val1 becomes a reporter): ", txHash)

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // number of delegating reporters + 1 validator reporter

	// user0 tips 1trb for bch
	value := layerutil.EncodeValue(10000000.99)
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("user0", "oracle", "tip", user0Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped ", bchQId, "): ", txHash)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// user1 reports for bch spotprice
	txHash, err = val1.ExecTx(ctx, "user1", "oracle", "submit-value", user1Addr, bchQData, value, "--keyring-dir", val1.HomeDir())
	fmt.Println("TX HASH (user1 reported ", bchQId, "): ", txHash)
	require.NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// user1 unbonds all of their tokens
	txHash, err = val1.ExecTx(ctx, user1Addr, "staking", "unbond", val1valAddr, "1000000000loya", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user1 unbonds all of their tokens): ", txHash)

	// query staking module delegations
	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 2) // val1 and user0, user1 is unbonding

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power after unbonding: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1power.Sub(math.NewInt(1000*1e6))) // val1 power after delegations minus user1 unbonded amount

	// get unbondingBeforeDispute amount
	unbondingBeforeDispute, err := chain.StakingQueryUnbondingDelegations(ctx, reporters[1].Addr)
	require.NoError(err)
	require.Equal(unbondingBeforeDispute[0].Entries[0].Balance.String(), "1000000000")
	fmt.Println("unbonding before dispute: ", unbondingBeforeDispute)

	// query reporter module
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 reporters + 1 validator reporter

	// wait for query to expire and dispute from user0
	err = testutil.WaitForBlocks(ctx, 2, val1)
	require.NoError(err)
	microreport, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", user1Addr)
	require.NoError(err)
	var microReports e2e.ReportsResponse
	require.NoError(json.Unmarshal(microreport, &microReports))
	require.Equal(microReports.MicroReports[0].Reporter, user1Addr)
	require.Equal(microReports.MicroReports[0].Value, value)
	require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
	require.Equal(microReports.MicroReports[0].Power, "1000")
	require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")
	bz, err := json.Marshal(microReports.MicroReports[0])
	require.NoError(err)
	fmt.Println("bz: ", string(bz))

	// get user0 stake vefore resolving dispute
	user0StakingBeforeDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 staking before resolving dispute: ", user0StakingBeforeDispute)

	// dispute from user0
	decodedBytes, err := base64.StdEncoding.DecodeString(microReports.MicroReports[0].QueryID)
	require.NoError(err)
	hexStr := hex.EncodeToString(decodedBytes)
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "propose-dispute", microReports.MicroReports[0].Reporter, microReports.MicroReports[0].MetaId, hexStr, "major", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 opens a major dispute on user1): ", txHash)

	// query reporter module
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 pure reporters + 1 validator reporter
	// find the disputed reporter (user1) and verify they are jailed
	var disputedReporter *e2e.Reporter
	for _, reporter := range reportersRes.Reporters {
		if reporter.Address == reporters[1].Addr { // user1's address
			disputedReporter = reporter
			break
		}
	}
	require.NotNil(disputedReporter, "Disputed reporter not found")
	require.True(disputedReporter.Metadata.Jailed, "Disputed reporter should be jailed")
	require.Greater(disputedReporter.Metadata.JailedUntil, time.Now().Add(1000000*time.Hour)) // jailed over 100 years mua ha ha

	// check dispute status
	res, _, err = val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(res, &openDisputes))
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes)
	require.Equal(len(openDisputes.OpenDisputes.Ids), 1) // dispute 1 is open

	// vote from user0 (all tipping power)
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 votes support for dispute 1): ", txHash)
	// vote from team (should be at least 66% voting power after (33% from team, 33% from user group))
	txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (team votes support for dispute 1): ", txHash)

	// check dispute status
	res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes
	require.NoError(json.Unmarshal(res, &disputes))
	fmt.Println("disputes: ", disputes)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 2)  // should be resolved now
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "1") // stayed in first round
	expectedFeeTotal := (math.NewInt(1_000 * 1e6))                 // 100% of user0 power
	require.Equal(disputes.Disputes[0].Metadata.FeeTotal, expectedFeeTotal.String())
	expectedBurnAmount := (expectedFeeTotal).Quo(math.NewInt(20)) // 5% of total fee
	require.Equal(disputes.Disputes[0].Metadata.BurnAmount, expectedBurnAmount.String())
	require.Equal(disputes.Disputes[0].Metadata.SlashAmount, expectedFeeTotal.String()) // 1% of amt staked with val1 still
	require.Equal(disputes.Disputes[0].Metadata.InitialEvidence.Reporter, reporters[1].Addr)
	require.Equal(disputes.Disputes[0].Metadata.InitialEvidence.Value, value)

	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations to val1 before withdrawing fee refund ", i, ": ", delegations[i])
	}

	// query unbonding delegations for user0 before withdrawing fee refund, should be empty
	unbonding, err := chain.StakingQueryUnbondingDelegations(ctx, user0Addr)
	require.NoError(err)
	fmt.Println("unbonding delegations for user0: ", unbonding)

	// withdraw feerefund for user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "withdraw-fee-refund", user0Addr, "1", "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 withdraws fee refund): ", txHash)

	// check user0 stake after withdrawing fee refund, should contain 950 more trb
	user0StakingAfterWithdraw, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 delegation to val1 after withdrawing fee refund: ", user0StakingAfterWithdraw)
	require.Equal(user0StakingAfterWithdraw.Balance.Amount.String(), user0StakingBeforeDispute.Balance.Amount.Add(math.NewInt(950*1e6)).String())

	// check user0 free floating after withdraw fee refund, before claiming reward, should not change
	user0FreeFloatingBeforeClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating before claiming reward: ", user0FreeFloatingBeforeClaim)

	// claim reward for user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "claim-reward", "1", "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 claims reward): ", txHash)

	// check delegations after claiming reward
	delegationsRes, err := chain.StakingQueryDelegations(ctx, user0Addr)
	require.NoError(err)
	for i := range delegationsRes {
		fmt.Println("delegations by user0 after claiming reward ", i, ": ", delegationsRes[i])
	}
	require.Equal(len(delegationsRes), 1) // should be delegated to val1 only

	// check val1 delegations
	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations to val1 after claiming reward ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 2) // val1 and user0, user1 is gone

	// check user0 delegation to val1, should not have changed
	user0Delegation, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 delegation to val1: ", user0Delegation)
	require.Equal(user0Delegation.Balance.Amount.String(), user0StakingAfterWithdraw.Balance.Amount.String())

	// check user0 free floating after claiming reward
	user0FreeFloatingAfterClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating after claiming reward: ", user0FreeFloatingAfterClaim)
	require.Greater(user0FreeFloatingAfterClaim.Int64(), user0FreeFloatingBeforeClaim.Int64())
}

// reporter reports, their reporting power increases, then major dispute is opened on report with less power than they have now
func TestReportDelegateMoreMajorDispute(t *testing.T) {
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

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

	// make 2 users who will delegate to val1 and become reporters
	numReporters := 2
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	var user0Addr, user1Addr string
	var delegateAmt sdk.Coin
	for i := range numReporters {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt = sdk.NewCoin("loya", math.NewInt(1_000*1e6))
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
		if i == 0 {
			user0Addr = user.FormattedAddress()
		} else {
			user1Addr = user.FormattedAddress()
		}
	}

	// query staking module delegations
	delegations, err := chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, user1

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1StartPower.Add(expectedDelTotal))
	val1power := val1Staking.Tokens

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

	// all 2 delegators become reporters
	for i := range reporters {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// val1 becomes a reporter
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val1 becomes a reporter): ", txHash)

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // number of delegating reporters + 1 validator reporter

	// user0 tips 1trb for bch
	value := layerutil.EncodeValue(10000000.99)
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("user0", "oracle", "tip", user0Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped ", bchQId, "): ", txHash)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// user1 reports for bch spotprice
	txHash, err = val1.ExecTx(ctx, "user1", "oracle", "submit-value", user1Addr, bchQData, value, "--keyring-dir", val1.HomeDir())
	fmt.Println("TX HASH (user1 reported ", bchQId, "): ", txHash)
	require.NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// wait for query to expire
	err = testutil.WaitForBlocks(ctx, 2, val1)
	require.NoError(err)

	// get report to check reporter power
	res, _, err = val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", user1Addr)
	require.NoError(err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(res, &reports))
	fmt.Println("reports: ", reports)
	require.Equal(reports.MicroReports[0].Reporter, user1Addr)
	require.Equal(reports.MicroReports[0].Value, value)
	require.Equal(reports.MicroReports[0].Power, "1000")

	// user1 doubles their delegation
	txHash, err = val1.ExecTx(ctx, user1Addr, "staking", "delegate", val1valAddr, delegateAmt.String(), "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user1 delegates more): ", txHash)

	// user0 tips 1trb for bch again
	_, _, err = val1.Exec(ctx, val1.TxCommand("user0", "oracle", "tip", user0Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped ", bchQId, "): ", txHash)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// user1 reports for bch spotprice again
	txHash, err = val1.ExecTx(ctx, "user1", "oracle", "submit-value", user1Addr, bchQData, value, "--keyring-dir", val1.HomeDir())
	fmt.Println("TX HASH (user1 reported ", bchQId, "): ", txHash)
	require.NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// wait for query to expire
	err = testutil.WaitForBlocks(ctx, 2, val1)
	require.NoError(err)

	// get report to check reporter power for second report
	res, _, err = val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", user1Addr)
	require.NoError(err)
	require.NoError(json.Unmarshal(res, &reports))
	fmt.Println("reports: ", reports)
	require.Equal(reports.MicroReports[1].Reporter, user1Addr)
	require.Equal(reports.MicroReports[1].Value, value)
	require.Equal(reports.MicroReports[1].Power, "2000") // more power from second delegation

	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, and user1

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power after unbonding: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1power.Add(math.NewInt(1000*1e6))) // val1 power after initial delegations plus user1 addtl 1000 trb

	// query reporter module
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 reporters + 1 validator reporter

	// major dispute from user0
	microreport, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", user1Addr)
	require.NoError(err)
	var microReports e2e.ReportsResponse
	require.NoError(json.Unmarshal(microreport, &microReports))
	require.Equal(microReports.MicroReports[0].Reporter, user1Addr)
	require.Equal(microReports.MicroReports[0].Value, value)
	require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
	require.Equal(microReports.MicroReports[0].Power, "1000")
	require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")
	bz, err := json.Marshal(microReports.MicroReports[0])
	require.NoError(err)
	fmt.Println("bz: ", string(bz))

	// get user0 stake before resolving dispute
	user0StakingBeforeDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 staking before resolving dispute: ", user0StakingBeforeDispute)

	// get user1 stake before resolving dispute
	user1StakingBeforeDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 staking before resolving dispute: ", user1StakingBeforeDispute)

	// dispute from user0
	decodedBytes, err := base64.StdEncoding.DecodeString(microReports.MicroReports[0].QueryID)
	require.NoError(err)
	hexStr := hex.EncodeToString(decodedBytes)
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "propose-dispute", microReports.MicroReports[0].Reporter, microReports.MicroReports[0].MetaId, hexStr, "major", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 opens a major dispute on user1): ", txHash)

	// query reporter module
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 pure reporters + 1 validator reporter
	// find the disputed reporter (user1) and verify they are jailed
	var disputedReporter *e2e.Reporter
	for _, reporter := range reportersRes.Reporters {
		if reporter.Address == reporters[1].Addr { // user1's address
			disputedReporter = reporter
			break
		}
	}
	require.NotNil(disputedReporter, "Disputed reporter not found")
	require.True(disputedReporter.Metadata.Jailed, "Disputed reporter should be jailed")
	require.Greater(disputedReporter.Metadata.JailedUntil, time.Now().Add(1000000*time.Hour)) // jailed over 100 years mua ha ha

	// check dispute status
	res, _, err = val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(res, &openDisputes))
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes)
	require.Equal(len(openDisputes.OpenDisputes.Ids), 1) // dispute 1 is open

	// vote from user0 (all tipping power)
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 votes support for dispute 1): ", txHash)
	// vote from team (should be at least 66% voting power after (33% from team, 33% from user group))
	txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (team votes support for dispute 1): ", txHash)

	// check dispute status
	res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes
	require.NoError(json.Unmarshal(res, &disputes))
	fmt.Println("disputes: ", disputes)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 2)  // should be resolved now
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "1") // stayed in first round
	expectedFeeTotal := (math.NewInt(1_000 * 1e6))                 // 100% of user0 power at time of report
	require.Equal(disputes.Disputes[0].Metadata.FeeTotal, expectedFeeTotal.String())
	expectedBurnAmount := (expectedFeeTotal).Quo(math.NewInt(20)) // 5% of total fee
	require.Equal(disputes.Disputes[0].Metadata.BurnAmount, expectedBurnAmount.String())
	require.Equal(disputes.Disputes[0].Metadata.SlashAmount, expectedFeeTotal.String())
	require.Equal(disputes.Disputes[0].Metadata.InitialEvidence.Reporter, reporters[1].Addr)
	require.Equal(disputes.Disputes[0].Metadata.InitialEvidence.Value, value)

	// check on disputed reporter again
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 pure reporters + 1 validator reporter
	fmt.Println("reportersRes: ", reportersRes)
	// find the disputed reporter (user1) and verify they are jailed
	for _, reporter := range reportersRes.Reporters {
		if reporter.Address == reporters[1].Addr { // user1's address
			disputedReporter = reporter
			break
		}
	}
	require.NotNil(disputedReporter, "Disputed reporter not found")
	require.True(disputedReporter.Metadata.Jailed, "Disputed reporter should be jailed")
	require.Greater(disputedReporter.Metadata.JailedUntil, time.Now().Add(1000000*time.Hour)) // jailed over 100 years mua ha ha

	// get user1 stake after resolving dispute
	user1StakingAfterDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 staking after resolving dispute: ", user1StakingAfterDispute)
	require.Equal(user1StakingAfterDispute.Balance.Amount.String(), user1StakingBeforeDispute.Balance.Amount.Sub(expectedFeeTotal).String()) // only slashed power at time of report (1000 trb)

	// withdraw feerefund for user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "withdraw-fee-refund", user0Addr, "1", "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 withdraws fee refund): ", txHash)

	// check user0 stake after withdrawing feerefund, should contain 950 more trb
	user0StakingAfterWithdraw, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 delegation to val1 after withdrawing fee refund: ", user0StakingAfterWithdraw)
	require.Equal(user0StakingAfterWithdraw.Balance.Amount.String(), user0StakingBeforeDispute.Balance.Amount.Add(math.NewInt(950*1e6)).String())

	// check user0 free floating after withdraw fee refund, before claiming reward, should not change
	user0FreeFloatingBeforeClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating before claiming reward: ", user0FreeFloatingBeforeClaim)

	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations to val1 before withdrawing fee refund ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, and user1

	// claim reward for user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "claim-reward", "1", "--keyring-dir", val1.HomeDir(), "--gas", "500000", "--fees", "10loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 claims reward): ", txHash)

	// check delegations after claiming reward
	delegationsRes, err := chain.StakingQueryDelegations(ctx, user0Addr)
	require.NoError(err)
	for i := range delegationsRes {
		fmt.Println("delegations by user0 after claiming reward ", i, ": ", delegationsRes[i])
	}
	require.Equal(len(delegationsRes), 1) // should be delegated to val1 only

	// check val1 delegations
	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations to val1 after claiming reward ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, and user1

	// check user0 delegation to val1, should not have changed
	user0Delegation, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 delegation to val1: ", user0Delegation)
	require.Equal(user0Delegation.Balance.Amount.String(), user0StakingAfterWithdraw.Balance.Amount.String())

	// check user0 free floating after claiming reward, should increase
	user0FreeFloatingAfterClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating after claiming reward: ", user0FreeFloatingAfterClaim)
	require.Greater(user0FreeFloatingAfterClaim.Int64(), user0FreeFloatingBeforeClaim.Int64())

	// check user1 delegation to val1, should be 1000 trb
	user1Delegation, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 delegation to val1: ", user1Delegation)
	require.Equal(user1Delegation.Balance.Amount.String(), "1000000000")

	// try to create reporter from user1
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to create reporter again): ", txHash)

	// check reporter module
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 pure reporters + 1 validator reporter

	// user1 tries to select another reporter, errors with selector already exists
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "select-reporter", user0Addr, "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to select another reporter): ", txHash)

	// user1 tries switching reporters, errors with cannot switch reporter if selector is a reporter
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "switch-reporter", user0Addr, "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to switch reporters): ", txHash)

	// user1 tries to remove self as selector, errors selector cannot be removed if it is the reporter's own address
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "remove-selector", user1Addr, "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to remove self as selector): ", txHash)

	// check reporter module
	res, _, err = val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // 2 pure reporters + 1 validator reporter
	fmt.Println("reportersRes: ", reportersRes)

	// user1 redelegates to val2
	txHash, err = val1.ExecTx(ctx, user1Addr, "staking", "redelegate", val1valAddr, val2valAddr, "1000000000loya", "--from", user1Addr, "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user1 redelegates to val2): ", txHash)

	// check on user1 delegation to val2
	user1Delegation, err = chain.StakingQueryDelegation(ctx, val2valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 delegation to val2: ", user1Delegation)
	require.Equal(user1Delegation.Balance.Amount.String(), "1000000000")

	// user1 tries to select another reporter, errors with selector already exists
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "select-reporter", user0Addr, "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to select another reporter): ", txHash)

	// user1 tries switching reporters, errors with cannot switch reporter if selector is a reporter
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "switch-reporter", user0Addr, "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to switch reporters): ", txHash)

	// user1 tries to remove self as selector, errors selector cannot be removed if it is the reporter's own address
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "remove-selector", user1Addr, "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 tries to remove self as selector): ", txHash)

	// user1 unbonds their second 1000 trb delegation from val2
	txHash, err = val1.ExecTx(ctx, user1Addr, "staking", "unbond", val2valAddr, "1000000000loya", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user1 unbonds their second 1000 trb delegation): ", txHash)

	// check unbonding delegations
	unbonding, err := chain.StakingQueryUnbondingDelegationsFrom(ctx, val2valAddr)
	require.NoError(err)
	fmt.Println("unbonding delegations from val2: ", unbonding)
	require.Equal(len(unbonding), 1)
	require.Equal(unbonding[0].DelegatorAddress, user1Addr)
	require.Equal(unbonding[0].ValidatorAddress, val2valAddr)
	require.Equal(unbonding[0].Entries[0].Balance.String(), "1000000000")
}

// 1% open, moves to 5%
// 2 open on same person for same report
// vote on both while both are open, resolve and check rewards
func TestEscalatingDispute(t *testing.T) {
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// get val1 staking power
	val1Staking, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	val1StartPower := val1Staking.Tokens
	fmt.Println("val1 staking power before delegations: ", val1StartPower)

	// make 2 users who will delegate to val1 and become reporters
	numReporters := 2
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	var user0Addr, user1Addr string
	var delegateAmt sdk.Coin
	for i := range numReporters {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt = sdk.NewCoin("loya", math.NewInt(1_000*1e6))
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
		if i == 0 {
			user0Addr = user.FormattedAddress()
		} else {
			user1Addr = user.FormattedAddress()
		}
	}

	// query staking module delegations
	delegations, err := chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, user1

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1StartPower.Add(expectedDelTotal))
	// val1power := val1Staking.Tokens

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

	// all 2 delegators become reporters
	for i := range reporters {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// val1 becomes a reporter
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val1 becomes a reporter): ", txHash)

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // number of delegating reporters + 1 validator reporter

	// user0 tips 1trb for bch
	value := layerutil.EncodeValue(10000000.99)
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("user0", "oracle", "tip", user0Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped ", bchQId, "): ", txHash)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// user1 reports for bch spotprice
	txHash, err = val1.ExecTx(ctx, "user1", "oracle", "submit-value", user1Addr, bchQData, value, "--keyring-dir", val1.HomeDir())
	fmt.Println("TX HASH (user1 reported ", bchQId, "): ", txHash)
	require.NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// wait for query to expire
	err = testutil.WaitForBlocks(ctx, 2, val1)
	require.NoError(err)

	// get report to check reporter power
	res, _, err = val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", user1Addr)
	require.NoError(err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(res, &reports))
	fmt.Println("reports: ", reports)
	require.Equal(reports.MicroReports[0].Reporter, user1Addr)
	require.Equal(reports.MicroReports[0].Value, value)
	require.Equal(reports.MicroReports[0].Power, "1000")

	// open warning dispute
	decodedBytes, err := base64.StdEncoding.DecodeString(reports.MicroReports[0].QueryID)
	require.NoError(err)
	hexStr := hex.EncodeToString(decodedBytes)
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "propose-dispute", reports.MicroReports[0].Reporter, reports.MicroReports[0].MetaId, hexStr, "warning", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 opens warning dispute): ", txHash)

	// check on dispute
	r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes
	err = json.Unmarshal(r, &disputes)
	require.NoError(err)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 1)   // open
	require.Equal(disputes.Disputes[0].Metadata.DisputeCategory, 1) // warning
	require.Equal(disputes.Disputes[0].Metadata.DisputeID, "1")     // open
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "1")
	require.Equal(disputes.Disputes[0].Metadata.FeeTotal, "10000000") // 10 * 1e6 is 1% of 1000
	fmt.Println("open dispute: ", disputes.Disputes[0])

	// try to open minor dispute on same report, errors with cannot jail already jailed reporter
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "propose-dispute", reports.MicroReports[0].Reporter, reports.MicroReports[0].MetaId, hexStr, "minor", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.Error(err)
	fmt.Println("TX HASH (user0 opens minor dispute): ", txHash)

	// user1 unjails reporter
	txHash, err = val1.ExecTx(ctx, user1Addr, "reporter", "unjail-reporter", reports.MicroReports[0].Reporter, "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user1 unjails reporter): ", txHash)

	// user0 opens minor dispute on same report
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "propose-dispute", reports.MicroReports[0].Reporter, reports.MicroReports[0].MetaId, hexStr, "minor", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 opens minor dispute): ", txHash)

	// check on dispute
	r, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	err = json.Unmarshal(r, &disputes)
	require.NoError(err)
	fmt.Println("disputes: ", disputes)
	require.Equal(disputes.Disputes[1].Metadata.DisputeStatus, 1)   // open, but now a minor dispute
	require.Equal(disputes.Disputes[1].Metadata.DisputeCategory, 2) // minor
	require.Equal(disputes.Disputes[1].Metadata.DisputeID, "2")     // open
	require.Equal(disputes.Disputes[1].Metadata.DisputeRound, "1")
	require.Equal(disputes.Disputes[1].Metadata.FeeTotal, "50000000") // 50 * 1e6 is 5% of 1000
	fmt.Println("open dispute: ", disputes.Disputes[1])

	// get user0 stake after proposing dispute
	user0Staking, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 stake after proposing dispute: ", user0Staking.Balance.Amount.String())
	expectedStake := math.NewInt(1000 * 1e6).Sub(math.NewInt(10 * 1e6)).Sub(math.NewInt(50 * 1e6))
	require.Equal(user0Staking.Balance.Amount.String(), expectedStake.String())

	// get user1 stake after proposing dispute
	user1Staking, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 stake after proposing dispute: ", user1Staking.Balance.Amount.String())
	require.Equal(user1Staking.Balance.Amount.String(), expectedStake.String())

	// resolve first dispute
	// vote from user0 on dispute 1
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 votes on dispute 1): ", txHash)

	// vote from user0 on dispute 2
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "vote", "2", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 votes on dispute 2): ", txHash)

	// vote from team on dispute 1
	txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (team votes on dispute 1): ", txHash)

	// wait 1 block for execution
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// make sure dispute 1 is resolved
	r, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	err = json.Unmarshal(r, &disputes)
	require.NoError(err)
	fmt.Println("disputes: ", disputes)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 2) // resolved
	// make sure dispute 2 is still open
	require.Equal(disputes.Disputes[1].Metadata.DisputeStatus, 1) // open

	// check user0 free floating balance before claiming
	user0BalanceBeforeClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating balance before claiming dispute 1 rewards: ", user0BalanceBeforeClaim)
	// claim dispute 1 rewards from user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "claim-reward", "1", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 claims dispute 1 rewards): ", txHash)
	// check user0 free floating balance, should get all of voting rewards (2.5% of dispute 1 fee)
	user0BalanceAfterClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating balance after claiming dispute 1 rewards: ", user0BalanceAfterClaim)
	require.Greater(user0BalanceAfterClaim.Int64(), user0BalanceBeforeClaim.Int64())
	require.Equal(user0BalanceAfterClaim.String(), user0BalanceBeforeClaim.Add(math.NewInt(250000)).String())

	// withdraw fee refund from user0 from dispute 1
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "withdraw-fee-refund", user0Addr, "1", "--gas", "250000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 withdraws fee refund from dispute 1): ", txHash)
	// check user0 stake, should get fee refund plus user1's dispute 1 slash amount
	user0StakingAfterRefund, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 stake after withdrawing fee refund from dispute 1: ", user0StakingAfterRefund.Balance.Amount.String())
	require.Greater(user0StakingAfterRefund.Balance.Amount.String(), user0Staking.Balance.Amount.String())
	require.Equal(user0StakingAfterRefund.Balance.Amount.String(), user0Staking.Balance.Amount.Add(math.NewInt(10*1e6)).Add(math.NewInt(95*1e5)).String())

	// vote from team and resolve dispute 2
	txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", "2", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (team votes on dispute 2): ", txHash)

	// wait 1 block for execution
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// withdraw fee refund from user0 from dispute 2
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "withdraw-fee-refund", user0Addr, "2", "--gas", "250000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 withdraws fee refund from dispute 2): ", txHash)

	// check user0 stake, should get fee refund plus user1's dispute 2 slash amount
	user0StakingAfterRefund2, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 stake after withdrawing fee refund from dispute 2: ", user0StakingAfterRefund2.Balance.Amount.String())
	require.Greater(user0StakingAfterRefund2.Balance.Amount.Int64(), user0StakingAfterRefund.Balance.Amount.Int64())
	require.Equal(user0StakingAfterRefund2.Balance.Amount.String(), user0StakingAfterRefund.Balance.Amount.Add(math.NewInt(50*1e6)).Add(math.NewInt(475*1e5)).String())

	// claim dispute 2 rewards from user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "claim-reward", "2", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 claims dispute 2 rewards): ", txHash)

	// check user0 free floating balance, should get all of voting rewards (2.5% of dispute 2 fee)
	user0BalanceAfterClaim2, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating balance after claiming dispute 2 rewards: ", user0BalanceAfterClaim2)
	require.Greater(user0BalanceAfterClaim2.Int64(), user0BalanceAfterClaim.Int64())
	require.Greater(user0BalanceAfterClaim2.Int64(), user0BalanceAfterClaim.Add(math.NewInt(1250000)).Int64()) // all of 2.5% of 50 trb, plus some dust from last claim
	require.Less(user0BalanceAfterClaim2.Int64(), user0BalanceAfterClaim.Add(math.NewInt(1251000)).Int64())    // less than 1000 loya in dust
}

// major dispute opened maliciously, disputer loses
func TestMajorDisputeAgainst(t *testing.T) {
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// get val1 staking power
	val1Staking, err := chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	val1StartPower := val1Staking.Tokens
	fmt.Println("val1 staking power before delegations: ", val1StartPower)

	// make 2 users who will delegate to val1 and become reporters
	numReporters := 2
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	var user0Addr, user1Addr string
	var delegateAmt sdk.Coin
	for i := range numReporters {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt = sdk.NewCoin("loya", math.NewInt(1_000*1e6))
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
		if i == 0 {
			user0Addr = user.FormattedAddress()
		} else {
			user1Addr = user.FormattedAddress()
		}
	}

	// query staking module delegations
	delegations, err := chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), 3) // val1, user0, user1

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1StartPower.Add(expectedDelTotal))
	// val1power := val1Staking.Tokens

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

	// all 2 delegators become reporters
	for i := range reporters {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// val1 becomes a reporter
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val1 becomes a reporter): ", txHash)

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // number of delegating reporters + 1 validator reporter

	// user0 tips 1trb for bch
	value := layerutil.EncodeValue(10000000.99)
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("user0", "oracle", "tip", user0Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped ", bchQId, "): ", txHash)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// user1 reports for bch spotprice
	txHash, err = val1.ExecTx(ctx, "user1", "oracle", "submit-value", user1Addr, bchQData, value, "--keyring-dir", val1.HomeDir())
	fmt.Println("TX HASH (user1 reported ", bchQId, "): ", txHash)
	require.NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// wait for query to expire
	err = testutil.WaitForBlocks(ctx, 2, val1)
	require.NoError(err)

	// get report to check reporter power
	res, _, err = val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", user1Addr)
	require.NoError(err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(res, &reports))
	fmt.Println("reports: ", reports)
	require.Equal(reports.MicroReports[0].Reporter, user1Addr)
	require.Equal(reports.MicroReports[0].Value, value)
	require.Equal(reports.MicroReports[0].Power, "1000")

	// get user0 stake before dispute
	user0StakingBeforeDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.NoError(err)
	fmt.Println("user0 stake before dispute: ", user0StakingBeforeDispute.Balance.Amount.String())

	// get user1 stake before dispute
	user1StakingBeforeDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 stake before dispute: ", user1StakingBeforeDispute.Balance.Amount.String())

	// open major dispute from user0
	decodedBytes, err := base64.StdEncoding.DecodeString(reports.MicroReports[0].QueryID)
	require.NoError(err)
	hexStr := hex.EncodeToString(decodedBytes)
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "propose-dispute", reports.MicroReports[0].Reporter, reports.MicroReports[0].MetaId, hexStr, "major", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 opens warning dispute): ", txHash)

	// check on dispute
	r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes
	err = json.Unmarshal(r, &disputes)
	require.NoError(err)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 1)   // open
	require.Equal(disputes.Disputes[0].Metadata.DisputeCategory, 3) // major
	require.Equal(disputes.Disputes[0].Metadata.DisputeID, "1")     // open
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "1")
	require.Equal(disputes.Disputes[0].Metadata.FeeTotal, "1000000000") // 1000 * 1e6 is 100% of 1000 trb
	fmt.Println("open dispute: ", disputes.Disputes[0])

	// there should be no delegations to val1 besides self now
	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	require.Equal(len(delegations), 1) // self only

	// vote from user0 against
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "vote", "1", "vote-against", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 votes against dispute 1): ", txHash)

	// vote from team against
	txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-against", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (team votes against dispute 1): ", txHash)

	// wait 1 block for execution
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	// check on dispute
	r, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	err = json.Unmarshal(r, &disputes)
	require.NoError(err)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 2)   // resolved
	require.Equal(disputes.Disputes[0].Metadata.DisputeCategory, 3) // major
	require.Equal(disputes.Disputes[0].Metadata.DisputeID, "1")
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "1")
	require.Equal(disputes.Disputes[0].Metadata.FeeTotal, "1000000000") // 1000 * 1e6 is 100% of 1000 trb

	// check on val1 delegations
	delegations, err = chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("delegations to val1 after dispute: ", delegations)
	require.Equal(len(delegations), 2) // self and user1 who got falsely disputed

	// check user1 delegation after dispute
	user1StakingAfterDispute, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 stake after dispute: ", user1StakingAfterDispute.Balance.Amount.String())
	expectedStake := math.NewInt(1000 * 1e6).Add(math.NewInt(950 * 1e6))
	require.Equal(user1StakingAfterDispute.Balance.Amount.String(), expectedStake.String())

	// attempted withdraw fee refund from user0, fails
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "withdraw-fee-refund", user0Addr, "1", "--gas", "250000", "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user0 withdraws fee refund): ", txHash)

	// check user0 stake after withdrawing refund, he should have lost everything
	_, err = chain.StakingQueryDelegation(ctx, val1valAddr, user0Addr)
	require.Error(err)

	// attempted withdraw fee refund from user1,  fails bc money was already sent
	txHash, err = val1.ExecTx(ctx, user1Addr, "dispute", "withdraw-fee-refund", user1Addr, "1", "--gas", "250000", "--keyring-dir", val1.HomeDir())
	require.Error(err)
	fmt.Println("TX HASH (user1 withdraws fee refund): ", txHash)

	// check user1 stake after withdrawing refund, still 1950 trb
	user1StakingAfterRefund, err := chain.StakingQueryDelegation(ctx, val1valAddr, user1Addr)
	require.NoError(err)
	fmt.Println("user1 stake after withdrawing fee refund: ", user1StakingAfterRefund.Balance.Amount.String())
	require.Equal(user1StakingAfterRefund.Balance.Amount.String(), expectedStake.String())

	// check user0 free floating balance before claiming
	user0BalanceBeforeClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating balance before claiming dispute 1 rewards: ", user0BalanceBeforeClaim)

	// claim rewards for user0
	txHash, err = val1.ExecTx(ctx, user0Addr, "dispute", "claim-reward", "1", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (user0 claims dispute 1 rewards): ", txHash)

	// check user0 free floating balance, should get all of voting rewards (2.5% of dispute 1 fee)
	user0BalanceAfterClaim, err := chain.BankQueryBalance(ctx, user0Addr, "loya")
	require.NoError(err)
	fmt.Println("user0 free floating balance after claiming dispute 1 rewards: ", user0BalanceAfterClaim)
	require.Greater(user0BalanceAfterClaim.Int64(), user0BalanceBeforeClaim.Int64())
	expectedBalance := user0BalanceBeforeClaim.Add(math.NewInt(25 * 1e6)) // 2.5% of 1000 trb
	require.Equal(user0BalanceAfterClaim.String(), expectedBalance.String())
}

// 2 out of 4 reporters submit, both are bad prices, dispute and unjail, then 4/4 submit bad prices, dispute and unjail
func TestEverybodyDisputed_NotConsensus_Consensus(t *testing.T) {
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

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

	// make 4 ppl who will delegate to val1 and become reporters
	numReporters := 4
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	var user0Addr, user1Addr, user2Addr, user3Addr string
	var delegateAmt sdk.Coin
	for i := range numReporters {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt = sdk.NewCoin("loya", math.NewInt(1_000*1e6))
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
		if i == 0 {
			user0Addr = user.FormattedAddress()
		} else if i == 1 {
			user1Addr = user.FormattedAddress()
		} else if i == 2 {
			user2Addr = user.FormattedAddress()
		} else if i == 3 {
			user3Addr = user.FormattedAddress()
		}
	}
	fmt.Println("user0Addr: ", user0Addr)
	fmt.Println("user1Addr: ", user1Addr)
	fmt.Println("user2Addr: ", user2Addr)
	fmt.Println("user3Addr: ", user3Addr)

	// query staking module delegations
	delegations, err := chain.StakingQueryDelegationsTo(ctx, val1valAddr)
	require.NoError(err)
	for i := range delegations {
		fmt.Println("delegations ", i, ": ", delegations[i])
	}
	require.Equal(len(delegations), numReporters+1) // val1, user0, user1, user2, user3

	// get val1 staking power
	val1Staking, err = chain.StakingQueryValidator(ctx, val1valAddr)
	require.NoError(err)
	fmt.Println("val1 staking power: ", val1Staking.Tokens)
	require.Equal(val1Staking.Tokens, val1StartPower.Add(expectedDelTotal))
	// val1power := val1Staking.Tokens

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

	// all 4 delegators become reporters
	for i := range reporters {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// val1 becomes a reporter
	txHash, err := val1.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.1", "1000000", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val1 becomes a reporter): ", txHash)

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // number of delegating reporters + 1 validator reporter

	// val 1 tips , 2/4 reporters submit, both are bad prices
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("validator", "oracle", "tip", val1Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped bch-usd): ", txHash)
	require.NoError(err)

	// 2/4 ppl submit, both are bad
	value := layerutil.EncodeValue(10000000.99)
	for i := range reporters[:2] {
		_, _, err = val1.Exec(ctx, val1.TxCommand(reporters[i].Keyname, "oracle", "submit-value", reporters[i].Addr, bchQData, value, "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " submitted bch-usd): ", txHash)
	}

	// wait for query to expire
	require.NoError(testutil.WaitForBlocks(ctx, 3, val1))

	// verify reports
	type UserReports struct {
		UserReport e2e.QueryMicroReportsResponse
		Timestamp  string
		qId        string
	}
	userReports := make([]UserReports, 2)
	for i := range reporters[:2] {
		res, _, err = val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", reporters[i].Addr)
		require.NoError(err)
		var userReport e2e.QueryMicroReportsResponse
		require.NoError(json.Unmarshal(res, &userReport))
		fmt.Println("userReport", i, ": ", userReport)
		require.Equal(len(userReport.MicroReports), 1)
		require.Equal(userReport.MicroReports[0].Reporter, reporters[i].Addr)
		require.Equal(userReport.MicroReports[0].Value, value)
		require.Equal(userReport.MicroReports[0].Power, "1000")
		decodedBytes, err := base64.StdEncoding.DecodeString(userReport.MicroReports[0].QueryID)
		require.NoError(err)
		hexStr := hex.EncodeToString(decodedBytes)
		userReports[i] = UserReports{
			UserReport: userReport,
			qId:        hexStr,
		}
		// get aggregate timestamp
		fmt.Println("getting aggregate timestamp for", hexStr, "...")
		res, _, err = val1.ExecQuery(ctx, "oracle", "get-current-aggregate-report", hexStr)
		require.NoError(err)
		var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
		err = json.Unmarshal(res, &currentAggRes)
		require.NoError(err)
		timestamp := currentAggRes.Timestamp
		userReports[i].Timestamp = timestamp
	}

	// open dispute on both reports from user3
	txHash, err = val1.ExecTx(ctx, user3Addr, "dispute", "propose-dispute", userReports[0].UserReport.MicroReports[0].Reporter, userReports[0].UserReport.MicroReports[0].MetaId, userReports[0].qId, "warning", "1000000000loya", "false", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (val1 proposed dispute on user0): ", txHash)
	txHash, err = val1.ExecTx(ctx, user3Addr, "dispute", "propose-dispute", userReports[1].UserReport.MicroReports[0].Reporter, userReports[1].UserReport.MicroReports[0].MetaId, userReports[1].qId, "warning", "1000000000loya", "false", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (val1 proposed dispute on user1): ", txHash)

	// assert there are 2 disputes open
	res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes
	require.NoError(json.Unmarshal(res, &disputes))
	require.Equal(len(disputes.Disputes), 2)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 1) // not resolved yet
	require.Equal(disputes.Disputes[1].Metadata.DisputeStatus, 1) // not resolved yet

	for i := range userReports {
		disputeId := strconv.Itoa(i + 1)
		// vote from val1 (all tipping power)
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val1 votes on dispute ", disputeId, "): ", txHash)

		// vote from val2 (0 power error)
		_, err = val2.ExecTx(ctx, "validator", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val2.HomeDir())
		require.Error(err)

		// check disputes status
		// should still be open bc only 33% of power has voted
		res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		require.NoError(json.Unmarshal(res, &disputes))
		fmt.Println("dispute 1: ", disputes.Disputes[i])
		require.Equal(disputes.Disputes[i].Metadata.DisputeStatus, 1) // not resolved yet

		// vote from team (should be at least 66% voting power after (33% from team, 33% from having one tip from val1))
		txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (team votes on dispute ", disputeId, "): ", txHash)

		// check on dispute status
		// should be resolved and executed
		r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		err = json.Unmarshal(r, &disputes)
		require.NoError(err)
		require.Equal(disputes.Disputes[i].Metadata.DisputeStatus, 2) // resolved now
		fmt.Println("resolved dispute ", disputes.Disputes[i].DisputeID)
	}

	// make sure aggregate is flagged
	res, _, err = val1.ExecQuery(ctx, "oracle", "retrieve-data", userReports[0].qId, userReports[0].Timestamp)
	require.NoError(err)
	var data e2e.QueryRetrieveDataResponse
	require.NoError(json.Unmarshal(res, &data))
	require.Equal(data.Aggregate.Flagged, true)
	res, _, err = val1.ExecQuery(ctx, "oracle", "retrieve-data", userReports[1].qId, userReports[1].Timestamp)
	require.NoError(err)
	var data2 e2e.QueryRetrieveDataResponse
	require.NoError(json.Unmarshal(res, &data2))
	require.Equal(data2.Aggregate.Flagged, true)

	// unjail reporters
	for i, usr := range userReports {
		txHash, err = val1.ExecTx(ctx, usr.UserReport.MicroReports[0].Reporter, "reporter", "unjail-reporter", usr.UserReport.MicroReports[0].Reporter, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (user", i, "unjails reporter): ", txHash)
	}

	// tip again, all 4 reporters submit bad prices
	tipAmt = math.NewInt(1_000_000)
	tip = sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("validator", "oracle", "tip", val1Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped bch-usd): ", txHash)
	require.NoError(err)

	// 4/4 reporters submit bad prices
	value = layerutil.EncodeValue(10000000.99)
	for i := range reporters {
		_, _, err = val1.Exec(ctx, val1.TxCommand(reporters[i].Keyname, "oracle", "submit-value", reporters[i].Addr, bchQData, value, "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " submitted bch-usd): ", txHash)
	}

	// wait for query to expire
	require.NoError(testutil.WaitForBlocks(ctx, 2, val1))

	// verify reports
	userReports = make([]UserReports, 4)
	for i := range reporters {
		res, _, err = val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", reporters[i].Addr)
		require.NoError(err)
		var userReport2 e2e.QueryMicroReportsResponse
		require.NoError(json.Unmarshal(res, &userReport2))
		fmt.Println("userReport", i, ": ", userReport2)
		if i < 2 {
			require.Equal(len(userReport2.MicroReports), 2)         // first 2 reporters should have 2 reports total
			require.Equal(userReport2.MicroReports[1].Power, "990") // first 2 lost 1% of their stake from previous dispute
			require.Equal(userReport2.MicroReports[1].Reporter, reporters[i].Addr)
			require.Equal(userReport2.MicroReports[1].Value, value)
		} else {
			require.Equal(len(userReport2.MicroReports), 1)          // last 2 reporters should have 1 report total
			require.Equal(userReport2.MicroReports[0].Power, "1000") // havent lost any stake yet
			require.Equal(userReport2.MicroReports[0].Reporter, reporters[i].Addr)
			require.Equal(userReport2.MicroReports[0].Value, value)
		}

		decodedBytes, err := base64.StdEncoding.DecodeString(userReport2.MicroReports[0].QueryID)
		require.NoError(err)
		hexStr := hex.EncodeToString(decodedBytes)
		userReports[i] = UserReports{
			UserReport: userReport2,
			qId:        hexStr,
		}
		// get aggregate timestamp
		fmt.Println("getting aggregate timestamp for", hexStr, "...")
		res, _, err = val1.ExecQuery(ctx, "oracle", "get-current-aggregate-report", hexStr)
		require.NoError(err)
		var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
		err = json.Unmarshal(res, &currentAggRes)
		require.NoError(err)
		timestamp := currentAggRes.Timestamp
		userReports[i].Timestamp = timestamp
	}

	// open dispute on all reports from user3
	for i := range userReports {
		if i < 2 {
			txHash, err = val1.ExecTx(ctx, val1Addr, "dispute", "propose-dispute", userReports[i].UserReport.MicroReports[1].Reporter, userReports[i].UserReport.MicroReports[1].MetaId, userReports[i].qId, "warning", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		} else {
			txHash, err = val1.ExecTx(ctx, val1Addr, "dispute", "propose-dispute", userReports[i].UserReport.MicroReports[0].Reporter, userReports[i].UserReport.MicroReports[0].MetaId, userReports[i].qId, "warning", "1000000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		}
		require.NoError(err)
		fmt.Println("TX HASH (val1 proposed dispute on user", i, "): ", txHash)
	}

	// assert there are 4 disputes open
	res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	require.NoError(json.Unmarshal(res, &disputes))
	require.Equal(len(disputes.Disputes), 6)
	require.Equal(disputes.Disputes[2].Metadata.DisputeStatus, 1) // not resolved yet
	require.Equal(disputes.Disputes[3].Metadata.DisputeStatus, 1) // not resolved yet
	require.Equal(disputes.Disputes[4].Metadata.DisputeStatus, 1) // not resolved yet
	require.Equal(disputes.Disputes[5].Metadata.DisputeStatus, 1) // not resolved yet

	for i := range userReports {
		disputeId := strconv.Itoa(i + 3) // disputes 3, 4, 5, 6
		// vote from val1 (all tipping power)
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val1 votes on dispute ", disputeId, "): ", txHash)

		// vote from val2 (0 power error)
		_, err = val2.ExecTx(ctx, "validator", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val2.HomeDir())
		require.Error(err)

		// check disputes status
		// should still be open bc only 33% of power has voted
		res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		require.NoError(json.Unmarshal(res, &disputes))
		fmt.Println("dispute ", i+3, ": ", disputes.Disputes[i+2])
		require.Equal(disputes.Disputes[i+2].Metadata.DisputeStatus, 1) // not resolved yet

		// vote from team (should be at least 66% voting power after (33% from team, 33% from having one tip from val1))
		txHash, err = val1.ExecTx(ctx, "team", "dispute", "vote", disputeId, "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (team votes on dispute ", disputeId, "): ", txHash)

		// check on dispute status
		// should be resolved and executed
		r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		err = json.Unmarshal(r, &disputes)
		require.NoError(err)
		require.Equal(disputes.Disputes[i+2].Metadata.DisputeStatus, 2) // resolved now
		fmt.Println("resolved dispute ", disputes.Disputes[i+2].DisputeID)
	}

	// make sure aggregate is flagged
	res, _, err = val1.ExecQuery(ctx, "oracle", "retrieve-data", userReports[3].qId, userReports[3].Timestamp)
	require.NoError(err)
	require.NoError(json.Unmarshal(res, &data))
	require.Equal(data.Aggregate.Flagged, true)
}

// add new query type, tip, report, dispute
func TestNewQueryTipReportDispute(t *testing.T) {
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
	teamMnemonic := "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
	require.NoError(chain.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

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

	// make 2 ppl who will delegate to val1 and become reporters
	numReporters := 2
	reporters := make([]ReporterAccs, numReporters)
	expectedDelTotal := math.NewInt(0)
	var user0Addr, user1Addr string
	var delegateAmt sdk.Coin
	for i := range numReporters {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt = sdk.NewCoin("loya", math.NewInt(1_000*1e6))
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
		if i == 0 {
			user0Addr = user.FormattedAddress()
		} else if i == 1 {
			user1Addr = user.FormattedAddress()
		}
	}
	fmt.Println("user0Addr: ", user0Addr)
	fmt.Println("user1Addr: ", user1Addr)

	// both users becomes reporters
	for i := range numReporters {
		commissRate := "0.1"
		minStakeAmt := "1000000"
		txHash, err := val1.ExecTx(ctx, reporters[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " becomes a reporter): ", txHash)
	}

	// user0 registers a new query
	queryType := "NFLSuperBowlChampion"
	spec := e2e.DataSpec{
		DocumentHash:      "legit-ipfs-hash!",
		ResponseValueType: "string",
		AggregationMethod: "weighted-mode",
		AbiComponents: []*registrytypes.ABIComponent{
			{
				Name:            "year of game",
				FieldType:       "string",
				NestedComponent: []*registrytypes.ABIComponent{},
			},
		},
		Registrar:         user0Addr,
		QueryType:         queryType,
		ReportBlockWindow: 10,
	}
	specBz, err := json.Marshal(spec)
	require.NoError(err)
	txHash, err := val1.ExecTx(ctx, user0Addr, "registry", "register-spec", queryType, string(specBz), "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (user0 registers a new query): ", txHash)

	// generate querydata
	queryBz, _, err := val1.ExecQuery(ctx, "registry", "generate-querydata", queryType, "[\"2025\"]")
	require.NoError(err)
	fmt.Println("queryBz: ", queryBz)
	var queryData e2e.QueryGenerateQuerydataResponse
	require.NoError(json.Unmarshal(queryBz, &queryData))
	fmt.Println("queryData: ", queryData)
	queryDataStr := hex.EncodeToString(queryData.QueryData)
	fmt.Println("queryDataStr: ", queryDataStr)

	// val1 tips the query
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand(user0Addr, "oracle", "tip", user0Addr, queryDataStr, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (val1 tips the query): ", txHash)

	// wait 1 block to prevent account sequence mismatch
	require.NoError(testutil.WaitForBlocks(ctx, 1, val1))

	// user0 and user1 report
	value := e2e.EncodeStringValue("Pittsburgh Steelers")
	fmt.Println("value: ", value)
	for i := range numReporters {
		txHash, err = val1.ExecTx(ctx, reporters[i].Keyname, "oracle", "submit-value", reporters[i].Addr, queryDataStr, value, "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (", reporters[i].Keyname, " reports the query): ", txHash)
	}

	// wait for query to expire
	require.NoError(testutil.WaitForBlocks(ctx, 10, val1))

	// verify reports
	type UserReports struct {
		UserReport e2e.QueryMicroReportsResponse
		Timestamp  string
		qId        string
	}
	userReports := make([]UserReports, numReporters)
	for i := range numReporters {
		var userReport e2e.QueryMicroReportsResponse
		res, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", reporters[i].Addr)
		require.NoError(err)
		require.NoError(json.Unmarshal(res, &userReport))
		fmt.Println("userReport: ", userReport)
		require.Equal(len(userReport.MicroReports), 1)
		reportedValue := userReport.MicroReports[0].Value
		fmt.Println("reportedValue: ", reportedValue)
		require.Equal(reportedValue, value)
		decodedVal, err := hex.DecodeString(reportedValue)
		require.NoError(err)
		fmt.Println("decodedVal: ", string(decodedVal))
		decodedBytes, err := base64.StdEncoding.DecodeString(userReport.MicroReports[0].QueryID)
		require.NoError(err)
		hexStr := hex.EncodeToString(decodedBytes)
		userReports[i] = UserReports{
			UserReport: userReport,
			qId:        hexStr,
		}
	}

	// verify aggregate
	res, _, err := val1.ExecQuery(ctx, "oracle", "get-current-aggregate-report", userReports[0].qId)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	require.NoError(json.Unmarshal(res, &currentAggRes))
	fmt.Println("currentAggRes: ", currentAggRes)
	require.Equal(currentAggRes.Aggregate.AggregatePower, "2000") // 2 reporters * 1000 power
	require.Equal(currentAggRes.Aggregate.AggregateValue, value)
	require.Equal(currentAggRes.Aggregate.Flagged, false)

	// val1 disputes both reports
	for i := range numReporters {
		txHash, err = val1.ExecTx(ctx, val1Addr, "dispute", "propose-dispute", userReports[i].UserReport.MicroReports[0].Reporter, userReports[i].UserReport.MicroReports[0].MetaId, userReports[i].qId, "warning", "1000000000loya", "false", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (val1 disputes report ", i, "): ", txHash)
	}
}
