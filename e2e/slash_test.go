package e2e_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestInactivitySlash tests the inactivity slashing mechanism
func TestInactivitySlash(t *testing.T) {
	require := require.New(t)

	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Use standard configuration
	chain, ic, ctx := e2e.SetupChain(t, 4, 0)
	defer ic.Close()

	// Get validators using the helper
	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)

	// Print validator info for debugging
	e2e.PrintValidatorInfo(ctx, validators)

	// Access specific validators that are used in the test
	val2 := validators[1].Node
	val4 := validators[3].Node
	val4valAddr := validators[3].ValAddr

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
