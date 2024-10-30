package e2e

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/cosmos/cosmos-sdk/types/module/testutil"
)

var (
	layerImageInfo = []ibc.DockerImage{
		{
			Repository: "localhost:5001/my-image2",
			Version:    "latest",
			UidGid:     "1000:1000",
		},
	}
	numValsOne       = 2
	numFullNodesZero = 0

	baseBech32 = "tellor"
)

// This test is meant to be used as a basic interchaintest tutorial.
// Code snippets are broken down in ./docs/upAndRunning.md
func LayerSpinup(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	cosmos.SetSDKConfig(baseBech32)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		LayerChainSpec(numValsOne, numFullNodesZero, "layer-1"),
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	layer := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(layer)

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
}

func LayerChainSpec(nv, nf int, chainId string) *interchaintest.ChainSpec {
	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
	}
	return &interchaintest.ChainSpec{
		NumValidators: &nv,
		NumFullNodes:  &nf,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           "layer",
			ChainID:        chainId,
			Bin:            "layerd",
			Denom:          "loya",
			Bech32Prefix:   "tellor",
			CoinType:       "118",
			GasPrices:      "0.000025loya",
			GasAdjustment:  1.1,
			TrustingPeriod: "504h",
			NoHostMount:    false,
			Images:         layerImageInfo,
			EncodingConfig: LayerEncoding(),
			ModifyGenesis:  cosmos.ModifyGenesis(modifyGenesis),
		},
	}
}

func LayerEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	return &cfg
}
