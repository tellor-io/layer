package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/relayer"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
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

func TestIbcInterchainQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	cosmos.SetSDKConfig("tellor")

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	ctx := context.Background()

	client, network := interchaintest.DockerSetup(t)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		e2e.LayerChainSpec(1, 0, "layer-1"),
		e2e.LayerChainSpec(1, 0, "layer-2"),
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	layer1, layer2 := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "100"),
	).Build(t, client, network)

	const pathName = "layer1-layer2"
	const relayerName = "relayer"

	ic := interchaintest.NewInterchain().
		AddChain(layer1).
		AddChain(layer2).
		AddRelayer(r, relayerName).
		AddLink(interchaintest.InterchainLink{
			Chain1:  layer1,
			Chain2:  layer2,
			Relayer: r,
			Path:    pathName,
			CreateChannelOpts: ibc.CreateChannelOptions{
				SourcePortName: "oracle",
				DestPortName:   "icqhost",
				Order:          ibc.Unordered,
				Version:        "icq-1",
			},
		})

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,

		SkipPathCreation: false,
	}))

	t.Cleanup(func() {
		_ = ic.Close()
	})

	layer2validator := layer2.Validators[0]

	valAddress, err := layer2validator.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)

	// create reporter
	_, err = layer2.GetNode().ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "--keyring-dir", layer2.HomeDir())
	require.NoError(t, err)

	// tip query
	_, err = layer2validator.ExecTx(ctx, "validator", "oracle", "tip", valAddress, qData, "1000000loya", "--keyring-dir", layer2.HomeDir())
	require.NoError(t, err)

	// submit value
	_, err = layer2validator.ExecTx(ctx, "validator", "oracle", "submit-value", valAddress, qData, value, "--keyring-dir", layer2.HomeDir())
	require.NoError(t, err)

	// Query for the recently created channel-id.
	channels, err := r.GetChannels(ctx, eRep, layer1.Config().ChainID)
	require.NoError(t, err)

	// Start the relayer and set the cleanup function.
	err = r.StartRelayer(ctx, eRep, pathName)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := r.StopRelayer(ctx, eRep)
		if err != nil {
			t.Logf("an error occurred while stopping the relayer: %s", err)
		}
	})

	err = testutil.WaitForBlocks(ctx, 5, layer1, layer2)
	require.NoError(t, err)

	chanID := channels[0].Counterparty.ChannelID
	require.NotEmpty(t, chanID)

	// get aggreate report
	qidbz, err := utils.QueryIDFromDataString(qData)
	require.NoError(t, err)

	cmd := []string{
		"layerd", "tx", "oracle", "send-query-get-current-aggregated-report", chanID, hex.EncodeToString(qidbz),
		"--node", layer1.GetRPCAddress(),
		"--home", layer1.HomeDir(),
		"--chain-id", layer1.Config().ChainID,
		"--from", "validator",
		"--keyring-dir", layer1.HomeDir(),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}

	// Execute interchain query
	stdout, _, err := layer1.Exec(ctx, cmd, nil)
	require.NoError(t, err)
	fmt.Println(string(stdout))

	err = testutil.WaitForBlocks(ctx, 10, layer1, layer2)
	require.NoError(t, err)
	cmd = []string{
		"layerd", "query", "oracle", "query-state", "--sequence", "1",
		"--node", layer1.GetRPCAddress(),
		"--home", layer1.HomeDir(),
		"--chain-id", layer1.Config().ChainID,
		"--output", "json",
	}

	response, _, err := layer1.Exec(ctx, cmd, nil)
	fmt.Println(string(response), "response")
	require.NoError(t, err)
	// Validate response
	var aggReport AggregateReport
	err = json.Unmarshal(response, &aggReport)
	require.NoError(t, err)
	fmt.Println("Aggregate report: ", aggReport)
	require.Equal(t, aggReport.Aggregate.AggregateReporter, valAddress)

}
