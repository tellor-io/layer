package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

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

func TestUpdateTeamAddr(t *testing.T) {
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

	// val0 calls update team addr, tries to make val1 team addr, should err bc signer is not team
	newTeamAddrBad := validators[1].ValAddr
	txHash, err := validators[0].Val.ExecTx(ctx, "validator", "dispute", "update-team", newTeamAddrBad, "--fees", "25loya", "--keyring-dir", validators[0].Val.HomeDir(), "--chain-id", chain.Config().ChainID)
	require.ErrorContains(err, "expected teamaccount as only signer for updateTeam message")
	fmt.Println("TX HASH(Update Team Addr called (bad signer)): ", txHash)

	// team calls update team addr, make val1 val addr the team addr, should error with bech32  expected tellor got tellorvaloper
	txHash, err = validators[0].Val.ExecTx(ctx, "team", "dispute", "update-team", newTeamAddrBad, "--fees", "25loya", "--keyring-dir", validators[0].Val.HomeDir(), "--chain-id", chain.Config().ChainID)
	require.ErrorContains(err, "invalid Bech32 prefix;")
	fmt.Println("TX HASH(Update Team Addr called (bad new addr)): ", txHash)

	// team calls update team addr, make val1 addr the team addr, all square
	newTeamAddrGood := validators[1].Addr
	txHash, err = validators[0].Val.ExecTx(ctx, "team", "dispute", "update-team", newTeamAddrGood, "--fees", "25loya", "--keyring-dir", validators[0].Val.HomeDir(), "--chain-id", chain.Config().ChainID)
	require.NoError(err)
	fmt.Println("TX HASH(Update Team Addr called (success)): ", txHash)

	// query team addr to confirm update
	teamAddrBz, _, err := validators[0].Val.ExecQuery(ctx, "dispute", "team-address")
	require.NoError(err)
	var teamAddr e2e.QueryTeamAddressResponse
	require.NoError(json.Unmarshal(teamAddrBz, &teamAddr))
	require.Equal(teamAddr.TeamAddress, newTeamAddrGood)
	height, err := validators[0].Val.Height(ctx)
	require.NoError(err)
	fmt.Println("current height: ", height)

	// wait 4 blocks
	require.NoError(testutil.WaitForBlocks(ctx, 4, validators[0].Val))
	height, err = validators[0].Val.Height(ctx)
	require.NoError(err)
	fmt.Println("current height: ", height)
}
