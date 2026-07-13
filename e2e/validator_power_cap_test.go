package e2e_test

import (
	"context"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	"cosmossdk.io/math"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// setupValidatorPowerCapChain runs four equal genesis validators under a 30% cap.
func setupValidatorPowerCapChain(t *testing.T) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	t.Helper()

	config := e2e.DefaultSetupConfig()
	config.NumValidators = 4
	config.ModifyGenesis = append(e2e.CreateStandardGenesis(),
		cosmos.NewGenesisKV(e2e.MaxValidatorPowerShareGenesisKey, "0.300000000000000000"),
	)
	return e2e.SetupChainWithCustomConfig(t, config)
}

func TestValidatorPowerCap(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := setupValidatorPowerCapChain(t)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	require.Len(validators, 4)

	bondedValidators, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Len(bondedValidators, 4)

	totalBonded := math.ZeroInt()
	for _, validator := range bondedValidators {
		totalBonded = totalBonded.Add(validator.Tokens)
	}

	// Under-cap redelegation (self-delegation keeps per-delegator delta at zero).
	_, err = validators[1].Node.ExecTx(ctx, validators[1].AccAddr,
		"staking", "redelegate", validators[1].ValAddr, validators[0].ValAddr, "1loya",
		"--keyring-dir", validators[1].Node.HomeDir(),
		"--gas", "500000", "--fees", "20loya",
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, validators[1].Node))

	// ~10% of bonded would push destination above 30%.
	overCapAmount := totalBonded.QuoRaw(10)
	require.True(overCapAmount.GT(totalBonded.QuoRaw(20)),
		"over-cap amount must exceed the 5%% total stake-change threshold headroom")
	_, err = validators[1].Node.ExecTx(ctx, validators[1].AccAddr,
		"staking", "redelegate", validators[1].ValAddr, validators[0].ValAddr,
		math.NewInt(overCapAmount.Int64()).String()+"loya",
		"--keyring-dir", validators[1].Node.HomeDir(),
		"--gas", "500000", "--fees", "20loya",
	)
	require.Error(err)
	require.ErrorContains(err, "validator bonded stake exceeds the max share of total bonded stake")
}
