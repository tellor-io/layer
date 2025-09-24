package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/relayer"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	"github.com/tellor-io/layer/utils"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestIbcInterchainQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cosmos.SetSDKConfig("tellor")

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

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
	nv := 1
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
						Repository: "layer-icq", // source code ibc branch
						Version:    "local",
						UIDGID:     "1025:1025",
					},
				},
				EncodingConfig:      e2e.LayerEncoding(),
				ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
				AdditionalStartArgs: []string{"--key-name", "validator"},
			},
		},
		{
			NumValidators: &nv,
			NumFullNodes:  &nf,
			ChainConfig: ibc.ChainConfig{
				Type:           "cosmos",
				Name:           "layer",
				ChainID:        "layer-receiver",
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
						UIDGID:     "1025:1025",
					},
				},
				EncodingConfig:      e2e.LayerEncoding(),
				ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
				AdditionalStartArgs: []string{"--key-name", "validator"},
			},
		},
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
			Chain1:           layer1,
			Chain2:           layer2,
			Relayer:          r,
			Path:             pathName,
			CreateClientOpts: ibc.CreateClientOptions{},
			CreateChannelOpts: ibc.CreateChannelOptions{
				SourcePortName: "oracle",
				DestPortName:   "icqhost",
				Order:          ibc.Unordered,
				Version:        "icq-1",
			},
		})

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))

	t.Cleanup(func() {
		_ = ic.Close()
	})

	layer2validator := layer2.Validators[0]

	valAddress, err := layer2validator.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)

	// create reporter
	_, err = layer2.GetNode().ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val2_moniker", "--keyring-dir", layer2.HomeDir())
	require.NoError(t, err)

	// tip query
	_, err = layer2validator.ExecTx(ctx, "validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", layer2.HomeDir())
	require.NoError(t, err)

	// submit value
	_, err = layer2validator.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", layer2.HomeDir())
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
		"--gas", "1000000",
		"--fees", "1000000loya",
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
	var aggReport e2e.AggregateReport
	err = json.Unmarshal(response, &aggReport)
	require.NoError(t, err)
	fmt.Println("Aggregate report: ", aggReport)
	require.Equal(t, aggReport.Aggregate.AggregateReporter, valAddress)
}
