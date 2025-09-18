package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetParams(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.Params.Set(ctx, params))

	p, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.EqualValues(t, params, p)
}

func TestGetAttestPenaltyTimeCutoff(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)

	// Test with default params (should be 0)
	err := k.Params.Set(ctx, types.DefaultParams())
	require.NoError(t, err)

	cutoff, err := k.GetAttestPenaltyTimeCutoff(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), cutoff)

	// Test with custom cutoff value
	customCutoff := uint64(1234567890000)
	params := types.DefaultParams()
	params.AttestPenaltyTimeCutoff = customCutoff
	err = k.Params.Set(ctx, params)
	require.NoError(t, err)

	cutoff, err = k.GetAttestPenaltyTimeCutoff(ctx)
	require.NoError(t, err)
	require.Equal(t, customCutoff, cutoff)
}

func TestGetMainnetChainId(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)

	// Test with default params (should be "tellor-1")
	err := k.Params.Set(ctx, types.DefaultParams())
	require.NoError(t, err)

	chainId, err := k.GetMainnetChainId(ctx)
	require.NoError(t, err)
	require.Equal(t, "tellor-1", chainId)

	// Test with custom chain ID
	customChainId := "custom-chain-1"
	params := types.DefaultParams()
	params.MainnetChainId = customChainId
	err = k.Params.Set(ctx, params)
	require.NoError(t, err)

	chainId, err = k.GetMainnetChainId(ctx)
	require.NoError(t, err)
	require.Equal(t, customChainId, chainId)
}
