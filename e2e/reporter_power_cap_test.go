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

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// setupPowerCapChain runs four equal genesis validators (25% of bonded stake
// each, under the 30% cap) and re-enables the reporter power cap that the
// standard e2e genesis disables.
func setupPowerCapChain(t *testing.T) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	t.Helper()

	config := e2e.DefaultSetupConfig()
	config.NumValidators = 4
	config.ModifyGenesis = append(e2e.CreateStandardGenesis(),
		cosmos.NewGenesisKV(e2e.MaxReporterPowerShareGenesisKey, "0.300000000000000000"),
	)
	return e2e.SetupChainWithCustomConfig(t, config)
}

func TestReporterPowerCap(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := setupPowerCapChain(t)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	require.Len(validators, 4)

	// validator 0 holds 25% of bonded stake, under the cap, so registering as
	// a reporter is allowed
	_, err = validators[0].Node.ExecTx(ctx, validators[0].AccAddr,
		"reporter", "create-reporter", "0.1", "1000000", "val0_moniker",
		"--keyring-dir", validators[0].Node.HomeDir(),
		"--gas", "500000", "--fees", "20loya",
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, validators[0].Node))

	// validator 1's account also holds 25% of bonded stake; selecting validator
	// 0's reporter would put that reporter at 50%, so the tx must be rejected
	_, err = validators[1].Node.ExecTx(ctx, validators[1].AccAddr,
		"reporter", "select-reporter", validators[0].AccAddr,
		"--keyring-dir", validators[1].Node.HomeDir(),
		"--gas", "500000", "--fees", "20loya",
	)
	require.Error(err)
	require.ErrorContains(err, "reporter power would reach or exceed the max share of total bonded stake")

	// a fresh account with a tiny bonded delegation keeps the reporter far
	// below the cap, so selecting is allowed
	user := interchaintest.GetAndFundTestUsers(t, ctx, "power-cap-user", math.NewInt(10_000_000), chain)[0]
	delegateAmt := sdk.NewCoin("loya", math.NewInt(1_000_000))
	_, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(),
		"staking", "delegate", validators[0].ValAddr, delegateAmt.String(),
		"--keyring-dir", validators[0].Node.HomeDir(),
		"--gas", "500000", "--fees", "20loya",
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, validators[0].Node))

	_, err = validators[0].Node.ExecTx(ctx, user.FormattedAddress(),
		"reporter", "select-reporter", validators[0].AccAddr,
		"--keyring-dir", validators[0].Node.HomeDir(),
		"--gas", "500000", "--fees", "20loya",
	)
	require.NoError(err)
}
