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
func TesUnbondMajorDispute(t *testing.T) {
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

	// query stkaing module delegations
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

	// query stkaing module delegations
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

	// check user0 stake after withdrawing feerefund, should contain 950 more trb
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

// reporter reports, thier reporting power increases, then major dispute is opened on report with less power than they have now
func TestGainTokensMajorDispute(t *testing.T) {

}

// 1% open, moves to 5%, moves to 100%
// 3 open on same person, different reports
func TestEscalatingDispute(t *testing.T) {

}
