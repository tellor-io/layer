package e2e_test

import (
	"context"
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
		fundAmt := math.NewInt(10_000 * 1e6)
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

	// get val1 staking power
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
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	for i, query := range queryDataList {
		_, _, err := val1.Exec(ctx, val1.TxCommand("validator", "oracle", "tip", val1Addr, query.QueryData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
		require.NoError(err)
		fmt.Println("val1 tipped ", query.QueryID)
		err = testutil.WaitForBlocks(ctx, 1, val1)
		require.NoError(err)

		txHash, err := val1.ExecTx(ctx, "validator", "oracle", "submit-value", reporters[i].Addr, query.QueryData, value, "--keyring-dir", val1.HomeDir())
		fmt.Println("TX HASH (", reporters[i].Keyname, " reported ", query.QueryID, "): ", txHash)
		require.NoError(err)

		// wait for query to expire and dispute
		err = testutil.WaitForBlocks(ctx, 2, val1)
		require.NoError(err)
		microreport, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", reporters[i].Addr)
		require.NoError(err)
		var microReports e2e.ReportsResponse
		require.NoError(json.Unmarshal(microreport, &microReports))
		// require.Equal(microReports.MicroReports[0].QueryID, query.QueryID) // unmarshalling type err ?
		require.Equal(microReports.MicroReports[0].Reporter, reporters[i].Addr)
		require.Equal(microReports.MicroReports[0].Value, value)
		require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
		require.Equal(microReports.MicroReports[0].Power, "1000")
		require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")
		bz, err := json.Marshal(microReports.MicroReports[0])
		require.NoError(err)
		fmt.Println("bz: ", string(bz))

		// dispute from validator
		txHash, err = val1.ExecTx(ctx, "validator", "dispute", "propose-dispute", string(bz), "warning", "500000000loya", "true", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (dispute on ", microReports.MicroReports[0].Reporter, "): ", txHash)

		// check dispute status
		// disputeId := i + 1
		// res, _, err := val1.ExecQuery(ctx, "dispute", "tally", "--dispute-id", strconv.Itoa(disputeId))
		// require.NoError(err)
		// var tally e2e.QueryDisputesTallyResponse
		// require.NoError(json.Unmarshal(res, &tally))
		// fmt.Println("tally: ", tally)
		// require.Equal(tally.Team.Support, "0")
		// require.Equal(tally.Team.Against, "0")
		// require.Equal(tally.Team.Invalid, "0")
		// require.Equal(tally.Users.VoteCount.Support, "0")
		// require.Equal(tally.Users.VoteCount.Against, "0")
		// require.Equal(tally.Users.VoteCount.Invalid, "0")
		// require.Equal(tally.Reporters.VoteCount.Support, "0")
		// require.Equal(tally.Reporters.VoteCount.Against, "0")
		// require.Equal(tally.Reporters.VoteCount.Invalid, "0")

	}

	// check open disputes
	res, _, err = val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(res, &openDisputes))
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes)
	require.Equal(len(openDisputes.OpenDisputes.Ids), 10) // all 10 disputes should be open

	// vote and resolve all disputes
	for i := 0; i < len(queryDataList); i++ {
		// vote from val1 (all tipping power)
		_, err = val1.ExecTx(ctx, "validator", "dispute", "vote", strconv.Itoa(i+1), "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)

		// vote from val2 (0 power error)
		_, err = val2.ExecTx(ctx, "validator", "dispute", "vote", strconv.Itoa(i+1), "vote-support", "--keyring-dir", val2.HomeDir())
		require.Error(err)

		// check dispute status
		res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		var disputes e2e.Disputes
		require.NoError(json.Unmarshal(res, &disputes))
		fmt.Println("disputes: ", disputes)
		require.Equal(disputes.Disputes[i].Metadata.DisputeStatus, 1) // not resolved yet

		// vote from team (should be at least 66% voting power after (33% from team, 33% from having one tip from val1))
		_, err = val1.ExecTx(ctx, "team", "dispute", "vote", strconv.Itoa(i+1), "vote-support", "--keyring-dir", val1.HomeDir())
		require.NoError(err)

		// check on dispute team vote
		teamVote, _, err := val1.ExecQuery(ctx, "dispute", "team-vote", strconv.Itoa(i+1))
		require.NoError(err)
		fmt.Println("Team address: ", string(teamVote))

		// check on dispute status
		r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
		require.NoError(err)
		err = json.Unmarshal(r, &disputes)
		require.NoError(err)
		fmt.Println("dispute ", i+1, ": ", disputes.Disputes[i])

		require.Equal(disputes.Disputes[i].Metadata.DisputeStatus, 2) // resolved now
	}
}

func TestFiveRounder(t *testing.T) {
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

	// queryValidators to confirm that validator is bonded
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
	for i := 0; i < numReporters; i++ {
		keyname := fmt.Sprintf("user%d", i)
		fundAmt := math.NewInt(10_000 * 1e6)
		delegateAmt := sdk.NewCoin("loya", math.NewInt(1_000*1e6))
		user := interchaintest.GetAndFundTestUsers(t, ctx, keyname, fundAmt, chain)[0]
		txHash, err := val1.ExecTx(ctx, user.FormattedAddress(), "staking", "delegate", val1valAddr, delegateAmt.String(), "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
		require.NoError(err)
		fmt.Println("TX HASH (", keyname, " delegates to val2): ", txHash)
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

	// all 2 delegators become reporters
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

	// query reporter module
	res, _, err := val1.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	fmt.Println("reporters res: ", reportersRes)
	require.Equal(len(reportersRes.Reporters), numReporters+1) // number of delegating reporters + 1 validator reporter

	// user0 tips 1trb and user1 reports for bch spotprice
	value := layerutil.EncodeValue(10000000.99)
	tipAmt := math.NewInt(1_000_000)
	tip := sdk.NewCoin("loya", tipAmt)
	_, _, err = val1.Exec(ctx, val1.TxCommand("user0", "oracle", "tip", reporters[0].Addr, bchQData, tip.String(), "--keyring-dir", val1.HomeDir()), val1.Chain.Config().Env)
	require.NoError(err)
	fmt.Println("TX HASH (user0 tipped ", bchQId, "): ", txHash)
	err = testutil.WaitForBlocks(ctx, 1, val1)
	require.NoError(err)

	txHash, err = val1.ExecTx(ctx, "user1", "oracle", "submit-value", reporters[1].Addr, bchQData, value, "--keyring-dir", val1.HomeDir())
	fmt.Println("TX HASH (user1 reported ", bchQId, "): ", txHash)
	require.NoError(err)

	// wait for query to expire and dispute from user0
	err = testutil.WaitForBlocks(ctx, 2, val1)
	require.NoError(err)
	microreport, _, err := val1.ExecQuery(ctx, "oracle", "get-reportsby-reporter", reporters[1].Addr)
	require.NoError(err)
	var microReports e2e.ReportsResponse
	require.NoError(json.Unmarshal(microreport, &microReports))
	// require.Equal(microReports.MicroReports[0].QueryID, query.QueryID) // unmarshalling type err ?
	require.Equal(microReports.MicroReports[0].Reporter, reporters[1].Addr)
	require.Equal(microReports.MicroReports[0].Value, value)
	require.Equal(microReports.MicroReports[0].AggregateMethod, "weighted-median")
	require.Equal(microReports.MicroReports[0].Power, "1000")
	require.Equal(microReports.MicroReports[0].QueryType, "SpotPrice")
	bz, err := json.Marshal(microReports.MicroReports[0])
	require.NoError(err)
	fmt.Println("bz: ", string(bz))

	// dispute from user0
	txHash, err = val1.ExecTx(ctx, reporters[0].Addr, "dispute", "propose-dispute", string(bz), "warning", "500000000loya", "false", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (dispute on ", microReports.MicroReports[0].Reporter, "): ", txHash)

	// check dispute status
	// disputeId := i + 1
	// res, _, err := val1.ExecQuery(ctx, "dispute", "tally", "--dispute-id", strconv.Itoa(disputeId))
	// require.NoError(err)
	// var tally e2e.QueryDisputesTallyResponse
	// require.NoError(json.Unmarshal(res, &tally))
	// fmt.Println("tally: ", tally)
	// require.Equal(tally.Team.Support, "0")
	// require.Equal(tally.Team.Against, "0")
	// require.Equal(tally.Team.Invalid, "0")
	// require.Equal(tally.Users.VoteCount.Support, "0")
	// require.Equal(tally.Users.VoteCount.Against, "0")
	// require.Equal(tally.Users.VoteCount.Invalid, "0")
	// require.Equal(tally.Reporters.VoteCount.Support, "0")
	// require.Equal(tally.Reporters.VoteCount.Against, "0")
	// require.Equal(tally.Reporters.VoteCount.Invalid, "0")

	// check open disputes
	res, _, err = val1.ExecQuery(ctx, "dispute", "open-disputes")
	require.NoError(err)
	var openDisputes e2e.QueryOpenDisputesResponse
	require.NoError(json.Unmarshal(res, &openDisputes))
	fmt.Println("openDisputes: ", openDisputes.OpenDisputes)
	require.Equal(len(openDisputes.OpenDisputes.Ids), 1) // dispute 1 is open

	// vote from user0 (all tipping power)
	_, err = val1.ExecTx(ctx, reporters[0].Addr, "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	require.NoError(err)

	// check dispute status
	res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes
	require.NoError(json.Unmarshal(res, &disputes))
	fmt.Println("disputes: ", disputes)
	fmt.Println("disputes[0] burn amount: ", disputes.Disputes[0].Metadata.BurnAmount)
	fmt.Println("disputes[0] dispute status: ", disputes.Disputes[0].Metadata.DisputeStatus)
	fmt.Println("disputes[0] dispute round: ", disputes.Disputes[0].Metadata.DisputeRound)
	fmt.Println("disputes[0] fee total: ", disputes.Disputes[0].Metadata.FeeTotal)
	fmt.Println("disputes[0] dispute end time: ", disputes.Disputes[0].Metadata.DisputeEndTime)
	fmt.Println("disputes[0] dispute start time: ", disputes.Disputes[0].Metadata.DisputeStartTime)
	fmt.Println("disputes[0] dispute id: ", disputes.Disputes[0].Metadata.DisputeID)
	fmt.Println("disputes[0] slash amt: ", disputes.Disputes[0].Metadata.SlashAmount)
	fmt.Println("disputes[0] voter reward: ", disputes.Disputes[0].Metadata.VoterReward)
	fmt.Println("disputes[0] category: ", disputes.Disputes[0].Metadata.DisputeCategory)
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 1)        // not resolved yet
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "1")       // still in first round
	expectedFeeTotal := (math.NewInt(1_000 * 1e6)).Quo(math.NewInt(100)) // 1% of amt staked with val1
	require.Equal(disputes.Disputes[0].Metadata.FeeTotal, expectedFeeTotal.String())
	expectedBurnAmount := (expectedFeeTotal).Quo(math.NewInt(20)) // 5% of total fee
	require.Equal(disputes.Disputes[0].Metadata.BurnAmount, expectedBurnAmount.String())
	require.Equal(disputes.Disputes[0].Metadata.SlashAmount, expectedFeeTotal.String()) // 1% of amt staked with val1 still
	require.Equal(disputes.Disputes[0].Metadata.InitialEvidence.Reporter, reporters[1].Addr)
	require.Equal(disputes.Disputes[0].Metadata.InitialEvidence.Value, value)
	require.Greater(disputes.Disputes[0].Metadata.DisputeEndTime, val1.CliContext().Viper.GetTime("validator"))

	// user0 opens a new round
	txHash, err = val1.ExecTx(ctx, reporters[0].Addr, "dispute", "propose-dispute", string(bz), "warning", "500000000loya", "false", "--keyring-dir", val1.HomeDir(), "--gas", "1000000", "--fees", "1000000loya")
	require.NoError(err)
	fmt.Println("TX HASH (second rd dispute on ", microReports.MicroReports[0].Reporter, "): ", txHash)

	res, _, err = val1.ExecQuery(ctx, "dispute", "disputes")
	require.NoError(err)
	fmt.Println("err: ", err)
	disputes = e2e.Disputes{}
	require.NoError(json.Unmarshal(res, &disputes))
	require.Equal(len(disputes.Disputes[0].DisputeID), 1)          // still one dispute open
	require.Equal(disputes.Disputes[0].Metadata.DisputeRound, "2") //
	require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 1)

	// vote from team (should be at least 66% voting power after (33% from team, 33% from having one tip from user0))
	// _, err = val1.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-support", "--keyring-dir", val1.HomeDir())
	// require.NoError(err)

	// // check on dispute team vote
	// teamVote, _, err := val1.ExecQuery(ctx, "dispute", "team-vote", "1")
	// require.NoError(err)
	// fmt.Println("Team address: ", string(teamVote))

	// // check on dispute status
	// r, _, err := val1.ExecQuery(ctx, "dispute", "disputes")
	// require.NoError(err)
	// err = json.Unmarshal(r, &disputes)
	// require.NoError(err)
	// fmt.Println("dispute: ", disputes.Disputes[0])

	// require.Equal(disputes.Disputes[0].Metadata.DisputeStatus, 2) // resolved now
}
