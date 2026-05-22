package e2e_test

import (
	"context"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// setupShareCapChain fixes the validator count for share-cap tests so the
// intended initial bonded distribution is explicit.
func setupShareCapChain(t *testing.T) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	t.Helper()

	config := e2e.DefaultSetupConfig()
	// Two equal genesis validators intentionally put each self-delegator above
	// 30%, so a tiny self-delegation isolates the share cap without hitting 5%.
	config.NumValidators = 2
	return e2e.SetupChainWithCustomConfig(t, config)
}

func TestDelegatorStakeShareLimit(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := setupShareCapChain(t)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	require.Len(validators, 2)

	bondedValidators, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	require.Len(bondedValidators, 2)

	totalBonded := math.ZeroInt()
	for _, validator := range bondedValidators {
		totalBonded = totalBonded.Add(validator.Tokens)
	}

	validator0, err := chain.StakingQueryValidator(ctx, validators[0].ValAddr)
	require.NoError(err)
	require.True(
		validator0.Tokens.MulRaw(10).GT(totalBonded.MulRaw(3)),
		"test requires validator 0 to start above the 30% bonded stake limit",
	)

	require.NoError(chain.SendFunds(ctx, "faucet", ibc.WalletAmount{
		Address: validators[0].AccAddr,
		Amount:  math.NewInt(1_000_000),
		Denom:   "loya",
	}))
	require.NoError(testutil.WaitForBlocks(ctx, 1, validators[0].Node))

	delegateAmt := sdk.NewCoin("loya", math.OneInt())
	require.True(
		delegateAmt.Amount.LT(totalBonded.QuoRaw(20)),
		"test amount must stay below the 5% total stake-change limit",
	)
	_, err = validators[0].Node.ExecTx(
		ctx,
		validators[0].AccAddr,
		"staking", "delegate",
		validators[0].ValAddr,
		delegateAmt.String(),
		"--keyring-dir", validators[0].Node.HomeDir(),
		"--gas", "500000",
		"--fees", "20loya",
	)
	require.Error(err)
	require.ErrorContains(err, "delegator bonded stake exceeds 30% of total bonded stake")
}

func TestShareCapAllows(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := setupShareCapChain(t)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	require.Len(validators, 2)

	// A fresh account with a tiny delegation is well below 30% of bonded stake.
	user := interchaintest.GetAndFundTestUsers(t, ctx, "share-cap-user", math.NewInt(1_000_000), chain)[0]
	delegateAmt := sdk.NewCoin("loya", math.OneInt())
	bondedValidators, err := chain.StakingQueryValidators(ctx, stakingtypes.BondStatusBonded)
	require.NoError(err)
	totalBonded := math.ZeroInt()
	for _, validator := range bondedValidators {
		totalBonded = totalBonded.Add(validator.Tokens)
	}
	require.True(
		delegateAmt.Amount.LT(totalBonded.QuoRaw(20)),
		"test amount must stay below the 5% total stake-change limit",
	)
	_, err = validators[0].Node.ExecTx(
		ctx,
		user.FormattedAddress(),
		"staking", "delegate",
		validators[0].ValAddr,
		delegateAmt.String(),
		"--keyring-dir", validators[0].Node.HomeDir(),
		"--gas", "500000",
		"--fees", "20loya",
	)
	require.NoError(err)
}
