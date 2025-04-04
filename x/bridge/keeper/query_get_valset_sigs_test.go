package keeper_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGetValsetSigs(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	getSigsResponse, err := keeper.NewQuerier(k).GetValsetSigs(ctx, nil)
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, getSigsResponse)

	getSigsResponse, err = keeper.NewQuerier(k).GetValsetSigs(ctx, &types.QueryGetValsetSigsRequest{})
	require.ErrorContains(t, err, "failed to get validator signatures")
	require.Nil(t, getSigsResponse)

	valset := types.NewBridgeValsetSignatures(3)
	valset.SetSignature(0, []byte("signature1"))
	valset.SetSignature(1, []byte("signature2"))
	valset.SetSignature(2, []byte("signature3"))

	err = k.BridgeValsetSignaturesMap.Set(ctx, 1, *valset)
	require.NoError(t, err)

	sigsHexExpected := make([]string, len(valset.Signatures))
	for i, sig := range valset.Signatures {
		sigsHexExpected[i] = common.Bytes2Hex(sig)
	}

	getSigsResponse, err = keeper.NewQuerier(k).GetValsetSigs(ctx, &types.QueryGetValsetSigsRequest{
		Timestamp: uint64(1),
	})
	require.NoError(t, err)
	require.NotNil(t, getSigsResponse)
	require.Equal(t, getSigsResponse.Signatures, sigsHexExpected)
}
