package e2e_test

import (
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"

	"cosmossdk.io/math"
)

func TestGas(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 4, 0)
	defer ic.Close()

	val1 := chain.Validators[0]
	val2 := chain.Validators[1]
	val3 := chain.Validators[2]
	val4 := chain.Validators[3]

	// all 4 vals become reporters
	_, err := chain.GetNode().ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val1_moniker", "--keyring-dir", val1.HomeDir())
	require.NoError(err)
	_, err = val2.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val2_moniker", "--keyring-dir", val2.HomeDir())
	require.NoError(err)
	_, err = val3.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val3_moniker", "--keyring-dir", val3.HomeDir())
	require.NoError(err)
	_, err = val4.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val4_moniker", "--keyring-dir", val4.HomeDir())
	require.NoError(err)

	// wait 1 block
	require.NoError(testutil.WaitForBlocks(ctx, 1, val1))

	// tip query
	_, _, err = val1.Exec(ctx, val1.TxCommand("validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", chain.HomeDir()), chain.Config().Env)
	require.NoError(err)

	t.Run("val1", func(t *testing.T) {
		t.Parallel()
		txHash, err := val1.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", val1.HomeDir())
		require.NoError(err)
		err = testutil.WaitForBlocks(ctx, 5, chain)
		require.NoError(err)
		resp, err := chain.GetNode().TxHashToResponse(ctx, txHash)
		require.NoError(err)
		fmt.Println("Tx hash: ", txHash)
		fmt.Println("Response: ", resp)
	})
	t.Run("val2", func(t *testing.T) {
		t.Parallel()
		txHash, err := val2.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", val2.HomeDir())
		require.NoError(err)
		err = testutil.WaitForBlocks(ctx, 5, chain)
		require.NoError(err)
		resp, err := chain.GetNode().TxHashToResponse(ctx, txHash)
		require.NoError(err)
		fmt.Println("Tx hash: ", txHash)
		fmt.Println("Response: ", resp)
	})
}
