package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
)

// SetupConfig holds configuration for test setup
type SetupConfig struct {
	NumValidators   int
	NumFullNodes    int
	ModifyGenesis   []cosmos.GenesisKV
	GasPrices       string
	GlobalFeeMinGas string
}

// DefaultSetupConfig returns standard test configuration
func DefaultSetupConfig() SetupConfig {
	fmt.Println("Using DefaultSetupConfig...")
	return SetupConfig{
		NumValidators:   2,
		NumFullNodes:    0,
		ModifyGenesis:   CreateStandardGenesis(),
		GasPrices:       "0.000025000000000000loya",
		GlobalFeeMinGas: "0.000025000000000000",
	}
}

// CreateStandardGenesis creates a standard genesis configuration
func CreateStandardGenesis() []cosmos.GenesisKV {
	teamAddressBytes := sdk.MustAccAddressFromBech32("tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf").Bytes()

	return []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", teamAddressBytes),
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "15s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "loya"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices.0.amount", "0.000025000000000000"),
		cosmos.NewGenesisKV("app_state.registry.dataspec.0.report_block_window", "5"),
	}
}

// ValidatorInfo contains validator node and address information
type ValidatorInfo struct {
	Node    *cosmos.ChainNode
	AccAddr string
	ValAddr string
}

// GetValidators retrieves all validators with their addresses
func GetValidators(ctx context.Context, chain *cosmos.CosmosChain) ([]ValidatorInfo, error) {
	var validators []ValidatorInfo

	for _, validator := range chain.Validators {
		accAddr, err := validator.AccountKeyBech32(ctx, "validator")
		if err != nil {
			return nil, fmt.Errorf("error getting validator account address: %w", err)
		}

		valAddr, err := validator.KeyBech32(ctx, "validator", "val")
		if err != nil {
			return nil, fmt.Errorf("error getting validator address: %w", err)
		}

		validators = append(validators, ValidatorInfo{
			Node:    validator,
			AccAddr: accAddr,
			ValAddr: valAddr,
		})
	}

	return validators, nil
}

// SetupTestChainWithConfig creates a test chain with the given configuration
func SetupChainWithCustomConfig(t *testing.T, config SetupConfig) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	fmt.Println("Setting up chain with custom config...")
	t.Helper()
	require := require.New(t)

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	time.Sleep(1 * time.Second)

	// Use the genesis configuration from config, or default if empty
	modifyGenesis := config.ModifyGenesis
	if modifyGenesis == nil {
		modifyGenesis = CreateStandardGenesis()
	}

	// Create chain spec
	chainSpec := &interchaintest.ChainSpec{
		NumValidators: &config.NumValidators,
		NumFullNodes:  &config.NumFullNodes,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           "layer",
			ChainID:        "layer",
			Bin:            "layerd",
			Denom:          "loya",
			Bech32Prefix:   "tellor",
			CoinType:       "118",
			GasPrices:      config.GasPrices,
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
			EncodingConfig:      LayerEncoding(),
			ModifyGenesis:       cosmos.ModifyGenesis(modifyGenesis),
			AdditionalStartArgs: []string{"--key-name", "validator"},
		},
	}

	// Create chains
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{chainSpec})

	client, network := interchaintest.DockerSetup(t)
	time.Sleep(1 * time.Second)

	layer := chains[0].(*cosmos.CosmosChain)
	ic := interchaintest.NewInterchain().AddChain(layer)

	ctx := context.Background()
	require.NoError(ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	time.Sleep(1 * time.Second)

	t.Cleanup(func() {
		_ = ic.Close()
		time.Sleep(1 * time.Second)
	})

	require.NoError(layer.RecoverKey(ctx, "team", teamMnemonic))
	require.NoError(layer.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: "tellor14ncp4jg0d087l54pwnp8p036s0dc580xy4gavf",
		Amount:  math.NewInt(1000000000000),
		Denom:   "loya",
	}))

	return layer, ic, ctx
}

// SetupStandardTestChain creates a test chain with standard configuration
func SetupChain(t *testing.T, numVals, numFullNodes int) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	fmt.Println("Setting up chain with standard configuration...")
	config := DefaultSetupConfig()
	config.NumValidators = numVals
	config.NumFullNodes = numFullNodes
	return SetupChainWithCustomConfig(t, config)
}

// PrintValidatorInfo prints validator information for debugging
func PrintValidatorInfo(ctx context.Context, validators []ValidatorInfo) {
	for i, validator := range validators {
		fmt.Printf("Validator %d:\n", i+1)
		fmt.Printf("  Account Address: %s\n", validator.AccAddr)
		fmt.Printf("  Validator Address: %s\n", validator.ValAddr)
		fmt.Printf("  Node: %s\n", validator.Node.Name())
	}
}

// CreateReporterFromValidator creates a reporter from a validator with stake
func CreateReporterFromValidator(ctx context.Context, validator ValidatorInfo, reporterName string, stakeAmount math.Int) (string, error) {
	txHash, err := validator.Node.ExecTx(ctx, "validator", "reporter", "create-reporter",
		"0.1", stakeAmount.String(), reporterName, "--keyring-dir", validator.Node.HomeDir())
	return txHash, err
}

// TipQuery tips a query with the specified amount
func TipQuery(ctx context.Context, validator *cosmos.ChainNode, queryData string, tipAmount math.Int) (string, error) {
	cmd := validator.TxCommand("validator", "oracle", "tip", queryData, tipAmount.String(), "--keyring-dir", validator.HomeDir())
	stdout, _, err := validator.Exec(ctx, cmd, validator.Chain.Config().Env)
	if err != nil {
		return "", err
	}

	// Parse the transaction output to get the tx hash
	var output cosmos.CosmosTx
	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return "", err
	}

	return output.TxHash, nil
}

// SubmitBatchReport submits a batch of reports
func SubmitBatchReport(ctx context.Context, validator *cosmos.ChainNode, reports []string, fees string) (string, error) {
	args := []string{"oracle", "batch-submit-value"}
	for _, report := range reports {
		args = append(args, "--values", report)
	}
	args = append(args, "--fees", fees, "--keyring-dir", validator.HomeDir())

	return validator.ExecTx(ctx, "validator", args...)
}
