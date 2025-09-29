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
)

// Use standard configuration
func Option1(t *testing.T) {
    
	chain, ic, ctx := SetupStandardTestChain(t, 4, 0)
	defer ic.Close()

    

	// Option 2: Use custom configuration
	// config := TestSetupConfig{
	//     NumValidators:    4,
	//     NumFullNodes:     0,
	//     VotingPeriod:     "20s",
	//     DepositPeriod:    "10s",
	//     ReportWindow:     "5",
	//     GasPrices:        "0.0025loya",
	//     GlobalFeeMinGas:  "0.0",
	// }
	// chain, ic, ctx := SetupTestChainWithConfig(t, config)
	// defer ic.Close()

	// Option 3: Use fast configuration for quick tests
	// chain, ic, ctx := SetupFastTestChain(t, 4, 0)
	// defer ic.Close()

	// Get validators using the new helper
	validators, err := GetValidators(ctx, chain)
	if err != nil {
		t.Fatalf("Failed to get validators: %v", err)
	}

	// Access validators easily
	val1 := validators[0].Node
	val2 := validators[1].Node
	val3 := validators[2].Node
	val4 := validators[3].Node

	// Get validator addresses
	val1Addr := validators[0].AccAddr
	val1ValAddr := validators[0].ValAddr

	// Print validator info for debugging
	PrintValidatorInfo(ctx, validators)
```