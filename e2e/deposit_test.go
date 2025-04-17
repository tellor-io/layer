package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// to run the test:
// in x/bridge/keeper/claim_deposit.go, change the 12 hr check to 2 min
// in x/registry/module/genesis.go, change the trbrbidge data spec ReportBlockWindow to 10
// in x/oracle/keeper/keeper.go, change the AutoClaimDeposit threshold to 2 min
// cd e2e
// go test -run TestDepositReport -timeout 10m
func TestDepositReport(t *testing.T) {
	require := require.New(t)

	t.Skip("")

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

	fileName := fmt.Sprintf("./dbfile/%s.db", t.Name())
	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		SkipPathCreation:  false,
		BlockDatabaseFile: fileName,
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
	for i := range validators {
		minStakeAmt := "1000000"
		moniker := fmt.Sprintf("reporter_moniker%d", i)
		txHash, err := validators[i].Val.ExecTx(ctx, validators[i].Addr, "reporter", "create-reporter", commissRate, minStakeAmt, moniker, "--keyring-dir", validators[i].Val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (validator", i, " becomes a reporter): ", txHash)
	}

	// validator tips bridge deposit id 1
	bridgeQueryDataString := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"
	tip := sdk.NewCoin("loya", math.NewInt(1*1e6))
	txHash, err := validators[0].Val.ExecTx(ctx, validators[0].Addr, "oracle", "tip", bridgeQueryDataString, tip.String(), "--keyring-dir", validators[0].Val.HomeDir())
	require.NoError(err)
	fmt.Println("TX HASH (val tips bridge deposit 1)", txHash)

	// both reporters report for the bridge deposit
	value := "0000000000000000000000003386518f7ab3eb51591571adbe62cf94540ead29000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000f424000000000000000000000000000000000000000000000000000000000000003e8000000000000000000000000000000000000000000000000000000000000002d74656c6c6f72317038386a7530796875746d6635703275373938787633756d616137756a77376763683972346600000000000000000000000000000000000000"
	for i := range validators {
		txHash, err := validators[i].Val.ExecTx(ctx, validators[i].Addr, "oracle", "submit-value", bridgeQueryDataString, value, "--keyring-dir", validators[0].Val.HomeDir())
		require.NoError(err)
		fmt.Println("TX HASH (validator", i, "reports bridge deposit 1)", txHash)
	}

	// make sure trbrbdige query is 10 blocks
	res, _, err := validators[0].Val.ExecQuery(ctx, "registry", "data-spec", "TRBBridge")
	require.NoError(err)
	var specRes e2e.QueryGetDataSpecResponse
	err = json.Unmarshal(res, &specRes)
	require.NoError(err)
	require.NotNil(specRes)
	fmt.Println("spec res: ", specRes.Spec)

	// wait 10 blocks for aggregate
	require.NoError(testutil.WaitForBlocks(ctx, 10, validators[0].Val))

	// verify aggregate
	queryDataBz, err := hex.DecodeString(bridgeQueryDataString)
	require.NoError(err)
	queryIdBz := utils.QueryIDFromData(queryDataBz)
	queryIdString := hex.EncodeToString(queryIdBz)
	res, _, err = validators[0].Val.ExecQuery(ctx, "oracle", "get-current-aggregate-report", queryIdString)
	require.NoError(err)
	var currentAggRes e2e.QueryGetCurrentAggregateReportResponse
	err = json.Unmarshal(res, &currentAggRes)
	require.NoError(err)
	require.NotNil(currentAggRes)

	// check if deposit claimed
	res, _, err = validators[0].Val.ExecQuery(ctx, "bridge", "get-deposit-claimed", "1")
	require.NoError(err)
	var claimedRes e2e.QueryGetDepositClaimedResponse
	err = json.Unmarshal(res, &claimedRes)
	require.NoError(err)
	require.False(claimedRes.Claimed)

	loyaHolders, err := chain.BankQueryDenomOwners(ctx, "loya")
	require.NoError(err)
	fmt.Println("Loya holders: ", loyaHolders)
	fmt.Println("len(loyaHolders): ", len(loyaHolders))

	// wait for 2 min window to expire, deposit should get claimed
	time.Sleep(120 * time.Second)

	// check if deposit claimed
	res, _, err = validators[0].Val.ExecQuery(ctx, "bridge", "get-deposit-claimed", "1")
	require.NoError(err)
	err = json.Unmarshal(res, &claimedRes)
	require.NoError(err)
	require.True(claimedRes.Claimed)

	// check destination balance
	loyaHolders, err = chain.BankQueryDenomOwners(ctx, "loya")
	require.NoError(err)
	fmt.Println("Loya holders: ", loyaHolders)
	fmt.Println("len(loyaHolders): ", len(loyaHolders))
}
