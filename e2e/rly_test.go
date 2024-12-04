package e2e_test

import (
	"context"
	"testing"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/tellor-io/layer/e2e"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"
)

func TestIbcInterchaintransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")
	client, network := interchaintest.DockerSetup(t)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		e2e.LayerChainSpec(1, 0, "layer-1"),
		e2e.LayerChainSpec(1, 0, "layer-2"),
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	layer1, layer2 := chains[0], chains[1]

	rf := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t))
	r := rf.Build(t, client, network)
	ibcPathName := "path"
	ic := interchaintest.NewInterchain().
		AddChain(layer1).
		AddChain(layer2).
		AddRelayer(r, "r").
		AddLink(interchaintest.InterchainLink{
			Chain1:  layer1,
			Chain2:  layer2,
			Relayer: r,
			Path:    ibcPathName,
		})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	ctx := context.Background()
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,

		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})
	w := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10), layer1, layer2)
	layer1Wallet, layer2Wallet := w[0], w[1]
	// layer1 -> layer2 channel info
	layer1ToLayer2ChannelInfo, err := r.GetChannels(ctx, eRep, layer1.Config().ChainID)
	require.NoError(t, err)
	layer1ToLayer2ChannelID := layer1ToLayer2ChannelInfo[0].ChannelID
	// layer2 -> layer1 channel info
	layer2ToLayer1ChannelInfo, err := r.GetChannels(ctx, eRep, layer2.Config().ChainID)
	require.NoError(t, err)
	layer2ToLayer1ChannelID := layer2ToLayer1ChannelInfo[0].ChannelID

	// loya IBC denom on layer2
	srcDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", layer2ToLayer1ChannelID, "loya"))
	dstIbcDenom := srcDenomTrace.IBCDenom()

	amountToSend := math.NewInt(5)
	transfer := ibc.WalletAmount{
		Address: layer2Wallet.FormattedAddress(),
		Denom:   "loya",
		Amount:  amountToSend,
	}
	_, err = layer1.SendIBCTransfer(ctx, layer1ToLayer2ChannelID, layer1Wallet.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	require.NoError(t, r.Flush(ctx, eRep, ibcPathName, layer1ToLayer2ChannelID))

	layer2Walletbal, err := layer2.GetBalance(ctx, layer2Wallet.FormattedAddress(), dstIbcDenom)
	require.NoError(t, err)
	require.Equal(t, transfer.Amount, layer2Walletbal)
}
