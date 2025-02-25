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

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestAttestation --timeout 5m

// user requests cycle list spot price attestation
func TestAttestation(t *testing.T) {
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
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
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

	type Validators struct {
		Addr    string
		ValAddr string
		Val     *cosmos.ChainNode
	}

	validators := make([]Validators, len(chain.Validators))
	for i, _ := range chain.Validators {
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

	// queryValidators to confirm that 2 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 2)

	// submit minting proposal and vote yes on it from all validators
	require.NoError(e2e.TurnOnMinting(ctx, chain, validators[0].Val))
	require.NoError(testutil.WaitForBlocks(ctx, 5, validators[0].Val))
	result, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(err)
	fmt.Println("Proposal status: ", result.Status.String())
	require.Equal(result.Status.String(), "PROPOSAL_STATUS_PASSED")

	// both validators become reporters
	for i, val := range chain.Validators {
		txHash, err := val.ExecTx(ctx, "validator", "reporter", "create-reporter", "0.5", "100000000", "--keyring-dir", val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (val", i+1, "becomes a reporter): ", txHash)
	}

	// query reporters
	res, _, err := validators[0].Val.ExecQuery(ctx, "reporter", "reporters")
	require.NoError(err)
	var reportersRes e2e.QueryReportersResponse
	err = json.Unmarshal(res, &reportersRes)
	require.NoError(err)
	require.Equal(len(reportersRes.Reporters), 2)

	// validator reporters report for the cycle list
	currentCycleListRes, _, err := validators[0].Val.ExecQuery(ctx, "oracle", "current-cyclelist-query")
	require.NoError(err)
	var currentCycleList e2e.QueryCurrentCyclelistQueryResponse
	err = json.Unmarshal(currentCycleListRes, &currentCycleList)
	require.NoError(err)
	fmt.Println("current cycle list: ", currentCycleList)
	for i, v := range validators {
		// report for the cycle list
		_, _, err = v.Val.Exec(ctx, v.Val.TxCommand("validator", "oracle", "submit-value", v.Addr, currentCycleList.QueryData, value, "--fees", "25loya", "--keyring-dir", v.Val.HomeDir()), v.Val.Chain.Config().Env)
		require.NoError(err)
		height, err := chain.Height(ctx)
		require.NoError(err)
		fmt.Println("validator [", i, "] reported at height ", height)
	}

	// wait for query to expire
	err = testutil.WaitForBlocks(ctx, 2, validators[0].Val)
	require.NoError(err)

	// check on reports
	var queryId1, queryId2 string
	var decodedQueryId string
	var timestamp string
	for i, v := range validators {
		reports, _, err := v.Val.ExecQuery(ctx, "oracle", "get-reportsby-reporter", v.Addr)
		require.NoError(err)
		var reportsRes e2e.QueryMicroReportsResponse
		err = json.Unmarshal(reports, &reportsRes)
		require.NoError(err)
		fmt.Println("reports from: ", v.Addr, ": ", reportsRes)
		require.Equal(len(reportsRes.MicroReports), 1) // each reporter should have one report
		if i == 0 {
			queryId1 = reportsRes.MicroReports[0].QueryID
		} else {
			queryId2 = reportsRes.MicroReports[0].QueryID
		}
		// decode queryId
		decodedBytes, err := base64.StdEncoding.DecodeString(reportsRes.MicroReports[0].QueryID)
		require.NoError(err)
		decodedQueryId = hex.EncodeToString(decodedBytes)
		fmt.Println("decodedQueryId: ", decodedQueryId)
		// parse timestamp
		parsedTime, err := time.Parse(time.RFC3339Nano, reportsRes.MicroReports[0].Timestamp)
		require.NoError(err)
		unixTimestamp := parsedTime.Unix()
		fmt.Println("unixTimestamp int64: ", unixTimestamp)
		timestamp = strconv.FormatInt(unixTimestamp, 10)
		fmt.Println("timestamp string: ", timestamp)
	}
	require.Equal(queryId1, queryId2) // make sure both reporters reported for the same query

	// request attestations for that report
	txHash, err := validators[0].Val.ExecTx(ctx, "validator", "bridge", "request-attestations", validators[0].Addr, decodedQueryId, timestamp, "--keyring-dir", validators[0].Val.HomeDir(), "--fees", "25loya")
	fmt.Println("TX HASH (val1 request attestation): ", txHash)
	require.NoError(err)

}
