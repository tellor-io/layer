package e2e_test

import (
	"context"
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
	"go.uber.org/zap/zaptest"
)

func TestGas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cosmos.SetSDKConfig("tellor")

	ctx := context.Background()

	client, network := interchaintest.DockerSetup(t)

	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.0"),
	}
	nv := 4
	nf := 0
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
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

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	layer1 := chains[0].(*cosmos.CosmosChain)
	ic := interchaintest.NewInterchain().
		AddChain(layer1)
	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,

		SkipPathCreation: false,
	}))

	t.Cleanup(func() {
		_ = ic.Close()
	})

	layer1validator := layer1.Validators[0]
	layer2validator := layer1.Validators[1]
	layer3validator := layer1.Validators[2]
	layer4validator := layer1.Validators[3]

	valAddress1, err := layer1validator.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	valAddress2, err := layer2validator.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)
	// valAddress3, err := layer3validator.AccountKeyBech32(ctx, "validator")
	// require.NoError(t, err)
	// valAddress4, err := layer4validator.AccountKeyBech32(ctx, "validator")
	// require.NoError(t, err)

	// create reporter
	_, err = layer1.GetNode().ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", layer1validator.HomeDir())
	require.NoError(t, err)
	_, err = layer2validator.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", layer2validator.HomeDir())
	require.NoError(t, err)
	_, err = layer3validator.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", layer3validator.HomeDir())
	require.NoError(t, err)
	_, err = layer4validator.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", layer4validator.HomeDir())
	require.NoError(t, err)

	// tip query
	_, err = layer1validator.ExecTx(ctx, "validator", "oracle", "tip", valAddress1, qData, "1000000loya", "--keyring-dir", layer1.HomeDir())
	require.NoError(t, err)

	t.Run("val1", func(t *testing.T) {
		t.Parallel()
		txHash, err := layer1validator.ExecTx(ctx, "validator", "oracle", "submit-value", valAddress1, qData, value, "--keyring-dir", layer1validator.HomeDir())
		require.NoError(t, err)
		err = testutil.WaitForBlocks(ctx, 5, layer1)
		require.NoError(t, err)
		resp, err := layer1.GetNode().TxHashToResponse(ctx, txHash)
		require.NoError(t, err)
		fmt.Println("Tx hash: ", txHash)
		fmt.Println("Response: ", resp)
	})
	t.Run("val2", func(t *testing.T) {
		t.Parallel()
		txHash, err := layer2validator.ExecTx(ctx, "validator", "oracle", "submit-value", valAddress2, qData, value, "--keyring-dir", layer2validator.HomeDir())
		require.NoError(t, err)
		err = testutil.WaitForBlocks(ctx, 5, layer1)
		require.NoError(t, err)
		resp, err := layer1.GetNode().TxHashToResponse(ctx, txHash)
		require.NoError(t, err)
		fmt.Println("Tx hash: ", txHash)
		fmt.Println("Response: ", resp)
	})
}
