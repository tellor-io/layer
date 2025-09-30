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

	// Use standard configuration
	chain, ic, ctx := e2e.SetupChain(t, 4, 0)
	defer ic.Close()

	layer1validator := chain.Validators[0]
	layer2validator := chain.Validators[1]
	layer3validator := chain.Validators[2]
	layer4validator := chain.Validators[3]

	_, err := layer1validator.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	_, err = layer2validator.AccountKeyBech32(ctx, "validator")
	require.NoError(err)
	// valAddress3, err := layer3validator.AccountKeyBech32(ctx, "validator")
	// require.NoError(err)
	// valAddress4, err := layer4validator.AccountKeyBech32(ctx, "validator")
	// require.NoError(t, err)

	// create reporter
	_, err = chain.GetNode().ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val1_moniker", "--keyring-dir", layer1validator.HomeDir())
	require.NoError(err)
	_, err = layer2validator.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val2_moniker", "--keyring-dir", layer2validator.HomeDir())
	require.NoError(err)
	_, err = layer3validator.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val3_moniker", "--keyring-dir", layer3validator.HomeDir())
	require.NoError(err)
	_, err = layer4validator.ExecTx(ctx, "validator", "reporter", "create-reporter", math.NewUint(0).String(), math.NewUint(1_000_000).String(), "val4_moniker", "--keyring-dir", layer4validator.HomeDir())
	require.NoError(err)

	// tip query
	_, err = layer1validator.ExecTx(ctx, "validator", "oracle", "tip", qData, "1000000loya", "--keyring-dir", chain.HomeDir())
	require.NoError(err)

	t.Run("val1", func(t *testing.T) {
		t.Parallel()
		txHash, err := layer1validator.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", layer1validator.HomeDir())
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
		txHash, err := layer2validator.ExecTx(ctx, "validator", "oracle", "submit-value", qData, value, "--keyring-dir", layer2validator.HomeDir())
		require.NoError(err)
		err = testutil.WaitForBlocks(ctx, 5, chain)
		require.NoError(err)
		resp, err := chain.GetNode().TxHashToResponse(ctx, txHash)
		require.NoError(err)
		fmt.Println("Tx hash: ", txHash)
		fmt.Println("Response: ", resp)
	})
}
