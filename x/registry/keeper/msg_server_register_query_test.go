package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestRegisterQuery(t *testing.T) {
	ms, ctx, k, _ := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// register a spec
	spec1 := types.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// register query for that spec
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

	// try to register query for unregsitered query type
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "unregisteredQueryType",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "query type not registered")
	require.Nil(t, registerQueryResult)

	//try to register mismatched datatype and datafields -- test all 256 combinations of data types now ?
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"a", "b"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "failed to encode arguments")
	require.Nil(t, registerQueryResult)

	// how would you arrive at err "failed to encode query data" ?
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "bool"},
		DataFields: []string{"c", "d"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "failed to encode arguments")
	require.Nil(t, registerQueryResult)

	// try to register query that already exists
	queryInput = &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err = ms.RegisterQuery(ctx, queryInput)
	require.ErrorContains(t, err, "query ID ", queryID1, " already exists")
	require.Nil(t, registerQueryResult)

}
