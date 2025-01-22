package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
)

var (
	layerImageInfo = []ibc.DockerImage{
		{
			Repository: "layer",
			Version:    "local",
			UidGid:     "1025:1025",
		},
	}
	numValsOne       = 2
	numFullNodesZero = 0

	baseBech32 = "tellor"

	teamMnemonic = "unit curious maid primary holiday lunch lift melody boil blossom three boat work deliver alpha intact tornado october process dignity gravity giggle enrich output"
)

func LayerSpinup(t *testing.T) *cosmos.CosmosChain {
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
	require.NoError(t, layer.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(t, layer.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	return layer
}

func LayerChainSpec(nv, nf int, chainId string) *interchaintest.ChainSpec {
	modifyGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
	}
	return &interchaintest.ChainSpec{
		NumValidators: &nv,
		NumFullNodes:  &nf,
		ChainConfig: ibc.ChainConfig{
			Type:                "cosmos",
			Name:                "layer",
			ChainID:             chainId,
			Bin:                 "layerd",
			Denom:               "loya",
			Bech32Prefix:        "tellor",
			CoinType:            "118",
			GasPrices:           "0.0025loya",
			GasAdjustment:       1.1,
			TrustingPeriod:      "504h",
			NoHostMount:         false,
			Images:              layerImageInfo,
			EncodingConfig:      LayerEncoding(),
			ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
			AdditionalStartArgs: []string{"--key-name", "validator", "--price-daemon-enabled=false"},
			PreGenesis:          pregenesis(),
		},
	}
}

func LayerEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	return &cfg
}

// for adding the secrets file required for bridging
func WriteSecretsFile(ctx context.Context, rpc, bridge string, tn *cosmos.ChainNode) error {
	secrets := []byte(`{
		"eth_rpc_url": "` + rpc + `",
		"token_bridge_contract": "` + bridge + `",
	}`)
	fmt.Println("Writing secrets file")
	return tn.WriteFile(ctx, secrets, "secrets.yaml")
}

func pregenesis() func(ibc.Chain) error {
	return func(chain ibc.Chain) error {
		layer := chain.(*cosmos.CosmosChain)
		for _, node := range layer.Validators {
			if err := WriteSecretsFile(context.Background(), "", "", node); err != nil {
				return err
			}
		}

		return nil
	}
}

// for unmarshalling the disputes response
type Disputes struct {
	Disputes []struct {
		DisputeID string   `json:"disputeId"`
		Metadata  Metadata `json:"metadata"`
	} `json:"disputes"`
}

type Metadata struct {
	HashID            string   `json:"hash_id"`
	DisputeID         string   `json:"dispute_id"`
	DisputeCategory   string      `json:"dispute_category"`
	DisputeFee        string   `json:"dispute_fee"`
	DisputeStatus     string      `json:"dispute_status"`
	DisputeStartTime  string   `json:"dispute_start_time"`
	DisputeEndTime    string   `json:"dispute_end_time"`
	DisputeStartBlock string   `json:"dispute_start_block"`
	DisputeRound      string   `json:"dispute_round"`
	SlashAmount       string   `json:"slash_amount"`
	BurnAmount        string   `json:"burn_amount"`
	InitialEvidence   Evidence `json:"initial_evidence"`
	FeeTotal          string   `json:"fee_total"`
	PrevDisputeIDs    []string `json:"prev_dispute_ids"`
	BlockNumber       string   `json:"block_number"`
	VoterReward       string   `json:"voter_reward"`
}

type Evidence struct {
	Reporter        string `json:"reporter"`
	Power           string `json:"power"`
	QueryType       string `json:"query_type"`
	QueryID         string `json:"query_id"`
	AggregateMethod string `json:"aggregate_method"`
	Value           string `json:"value"`
	Timestamp       string `json:"timestamp"`
	BlockNumber     string `json:"block_number"`
}
