package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestKeeper(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	//newK := k.NewKeeper(cdc, storeKey, memKey, paramStore)

	logger := k.Logger(types.UnwrapSDKContext(ctx))
	fmt.Println(logger)

	k.SetGenesisSpec(types.UnwrapSDKContext(ctx))
	k.SetGenesisQuery(types.UnwrapSDKContext(ctx))

	genesisSpec := k.GetGenesisSpec(types.UnwrapSDKContext(ctx))
	fmt.Println(genesisSpec)
	require.NotNil(t, genesisSpec)

	//how to access store ?

}
