package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestQueryGetQueryData(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)

	// register a spec and query
	spec1 := types.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)
	queryInput := &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err := ms.RegisterQuery(ctx, queryInput)
	require.NoError(t, err)
	require.NotNil(t, registerQueryResult)
	queryId := registerQueryResult.QueryId

	// call GetQueryData() with registered query, check that it equals what QueryData() returns as bytes
	retreivedQueryData, err := k.GetQueryData(ctx, &types.QueryGetQueryDataRequest{QueryId: queryId})
	queryData1 := retreivedQueryData.QueryData //query data from GetQueryData()
	require.NoError(t, err)
	queryData2, err := k.QueryData(unwrappedCtx, queryId) //query data from QueryData()
	require.NoError(t, err)
	require.Equal(t, queryData1, strings.ToLower(bytes.HexBytes(queryData2).String()))

	// call GetQueryData() with unregistered query
	retreivedQueryData, err = k.GetQueryData(ctx, &types.QueryGetQueryDataRequest{QueryId: "a"})
	fmt.Println(retreivedQueryData)
	require.Nil(t, retreivedQueryData)
	fmt.Println("err", err)
	require.ErrorContains(t, err, "invalid query id")

	require.False(t, keeper.IsQueryIdValid("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-=_+[]{}|;:'~,.<>?/"))

}

func TestIsQueryIdValid(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	//unwrappedCtx := sdk.UnwrapSDKContext(ctx)

	//65 characters
	require.False(t, keeper.IsQueryIdValid("0xabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789999"))
	require.False(t, keeper.IsQueryIdValid("0XabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789999"))
	//63 characters
	require.False(t, keeper.IsQueryIdValid("0xabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567899"))
	require.False(t, keeper.IsQueryIdValid("0XabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567899"))
	//0 characters
	require.False(t, keeper.IsQueryIdValid(""))
	require.False(t, keeper.IsQueryIdValid("0x"))
	require.False(t, keeper.IsQueryIdValid("0x                                                               "))
	//queryID can have symbols
	//require.False(t, keeper.IsQueryIdValid("0X000000000000000000000000000000000000000000000000000000000000000$"))
	//64 characters with 0x
	require.True(t, keeper.IsQueryIdValid("0xabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345678999"))
	require.True(t, keeper.IsQueryIdValid("0XabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345678999"))
	require.True(t, keeper.IsQueryIdValid("0x0000000000000000000000000000000000000000000000000000000000000000"))
	require.True(t, keeper.IsQueryIdValid("0X0000000000000000000000000000000000000000000000000000000000000000"))
}
