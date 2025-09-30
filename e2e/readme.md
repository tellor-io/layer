# E2E Tests

These are end to end tests using the [interchaintest](https://github.com/strangelove-ventures/interchaintest) framework. These tests spin up a live chain with a given number of nodes/validators in docker that you can run transactions and queries against.

## QUICK START

```sh
make docker-image
```

or

```sh
make get-heighliner
make local-image
```

## RUN TESTS
```
Run an individual test:
```sh
cd e2e
go test -v -run TestLayerFlow -timeout 10m
```

Note: do not reuse Layer Spinup 


## Example

```go
package e2e

import (
	"testing"
	"context"
	"github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
	// Basic usage - uses default configuration
	chain, ic, ctx := SetupChain(t, 2, 0)
	defer ic.Close()

	// Get validators using the helper
	validators, err := GetValidators(ctx, chain)
	require.NoError(t, err)

	// Access validators easily
	val1 := validators[0].Node
	val2 := validators[1].Node

	// Get validator addresses
	val1Addr := validators[0].AccAddr
	val1ValAddr := validators[0].ValAddr

	// Print validator info for debugging
	PrintValidatorInfo(ctx, validators)

	// Your test logic here...
	require.Equal(t, 2, len(validators))
	require.NotEmpty(t, val1Addr)
	require.NotEmpty(t, val1ValAddr)
}

func TestCustomConfiguration(t *testing.T) {
	// Custom gas prices only
	config := DefaultTestSetupConfig()
	config.GasPrices = "0.001loya"
	chain, ic, ctx := SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	// Custom genesis modifications
	customGenesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "30s"),
		cosmos.NewGenesisKV("app_state.dispute.params.team_address", customTeamAddress),
	}
	config = DefaultTestSetupConfig()
	config.ModifyGenesis = customGenesis
	chain, ic, ctx = SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	// Completely custom configuration
	config = TestSetupConfig{
		NumValidators:   4,
		NumFullNodes:    1,
		ModifyGenesis:   customGenesis,
		GasPrices:       "0.002loya",
		GlobalFeeMinGas: "0.001",
	}
	chain, ic, ctx = SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	// Your test logic here...
}
```