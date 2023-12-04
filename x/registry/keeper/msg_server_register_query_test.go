package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestRegisterQuery(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// register spec1
	spec1 := types.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// register query for spec1
	queryInput := &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err := ms.RegisterQuery(ctx, queryInput)
	require.NoError(t, err)
	require.NotNil(t, registerQueryResult)
	queryID1 := registerQueryResult.QueryId

	// Require Statements
	//
	// register query for unregistered query type
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "unregisteredQueryType",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "query type not registered")
	require.Nil(t, registerQueryResult)

	// register mismatched datatype and datafields -- test all 256 combinations of data types now ?
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"x", "y"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "failed to encode arguments")
	require.Nil(t, registerQueryResult)

	// how to get "failed to encode query data" ?
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "bytes"},
		DataFields: []string{"", ""},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "failed to encode arguments")
	//require.ErrorContains(t, err, "failed to encode query data")
	require.Nil(t, registerQueryResult)

	// register query that already exists
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "query ID ", queryID1, " already exists")
	require.Nil(t, registerQueryResult)

	// register empty data types
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"", ""},
		DataFields: []string{"", ""},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "failed to encode arguments")
	require.Nil(t, registerQueryResult)

}
