package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// cd e2e
// go test -run TestInactivitySlash --timeout 5m

// start with 4 validators, one of them goes offline
func TestInactivitySlash(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	// Use standard configuration with custom slashing parameter
	config := e2e.DefaultTestSetupConfig()
	config.NumValidators = 4
	config.NumFullNodes = 0

	// Add custom slashing parameter
	modifyGenesis := e2e.CreateStandardGenesis(config)
	modifyGenesis = append(modifyGenesis, cosmos.NewGenesisKV("app_state.slashing.params.signed_blocks_window", "4"))

	chain, ic, ctx := e2e.SetupChainWithCustomConfig(t, config)
	defer ic.Close()

	// Note: The standard setup already handles team key recovery and funding

	val1 := chain.Validators[0]
	val1Addr, err := val1.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val1valAddr, err := val1.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val1 Account Address: ", val1Addr)
	fmt.Println("val1 Validator Address: ", val1valAddr)
	val2 := chain.Validators[1]
	val2Addr, err := val2.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val2valAddr, err := val2.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val2 Account Address: ", val2Addr)
	fmt.Println("val2 Validator Address: ", val2valAddr)
	val4 := chain.Validators[3]
	val4Addr, err := val4.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	val4valAddr, err := val4.KeyBech32(ctx, "validator", "val")
	require.NoError(err)
	fmt.Println("val4 Account Address: ", val4Addr)
	fmt.Println("val4 Validator Address: ", val4valAddr)

	// queryValidators to confirm that 4 validators are bonded
	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Equal(len(vals), 4)

	// val 4 goes offline
	height, err := chain.Height(ctx)
	require.NoError(err)
	err = val4.PauseContainer(ctx)
	fmt.Println("paused val4 at height: ", height)
	require.NoError(err)

	// wait 5 blocks
	require.NoError(testutil.WaitForBlocks(ctx, 5, val2))

	// 4 validators, 1 jailed
	fmt.Println("querying validators...")
	valsQueryRes, _, err := e2e.QueryWithTimeout(ctx, val2, "staking", "validators")
	require.NoError(err)
	var validatorsRes e2e.QueryValidatorsResponse
	err = json.Unmarshal(valsQueryRes, &validatorsRes)
	require.NoError(err)
	fmt.Println("validatorsRes: ", validatorsRes)
	require.Equal(len(validatorsRes.Validators), 4)

	// make sure one validator is jailed
	jailedCount := 0
	for _, val := range validatorsRes.Validators {
		if val.Jailed {
			jailedCount++
		}
	}
	require.Equal(1, jailedCount, "expected exactly one validator to be jailed")

	// unpause val4
	err = val4.UnpauseContainer(ctx)
	require.NoError(err)
	height, err = chain.Height(ctx)
	require.NoError(err)
	fmt.Println("unpaused val4 at height: ", height)

	// wait 10 minutes
	time.Sleep(10 * time.Minute)

	// unjail val4
	txHash, err := val4.ExecTx(ctx, "validator", "slashing", "unjail", "--from", val4valAddr, "--keyring-dir", val4.HomeDir(), "--chain-id", val4.Chain.Config().ChainID)
	require.NoError(err)
	fmt.Println("unjailed val4 with tx hash: ", txHash)

	// make sure val4 is unjailed
	valsQueryRes, _, err = e2e.QueryWithTimeout(ctx, val2, "staking", "validators")
	require.NoError(err)
	var validatorsRes2 e2e.QueryValidatorsResponse
	err = json.Unmarshal(valsQueryRes, &validatorsRes2)
	require.NoError(err)
	fmt.Println("validatorsRes: ", validatorsRes2)
	require.Equal(len(validatorsRes2.Validators), 4)

	// make sure no validator is jailed
	jailedCount = 0
	for _, val := range validatorsRes2.Validators {
		if val.Jailed {
			jailedCount++
		}
	}
	require.Equal(0, jailedCount, "expected no validators to be jailed")
}
