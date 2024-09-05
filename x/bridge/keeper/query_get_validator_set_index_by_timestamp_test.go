package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestQuery_GetValidatorSetIndexByTimestamp(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	q := keeper.NewQuerier(k)

	testCases := []struct {
		name          string
		setup         func()
		req           *types.QueryGetValidatorSetIndexByTimestampRequest
		expectedIndex int64
		err           bool
	}{
		{
			name: "nil request",
			req:  nil,
			err:  true,
		},
		{
			name: "GetValidatorSetIndexByTimestamp not set",
			req: &types.QueryGetValidatorSetIndexByTimestampRequest{
				Timestamp: 100,
			},
			err: true,
		},
		{
			name: "success",
			setup: func() {
				err := k.ValsetTimestampToIdxMap.Set(ctx, 100, types.CheckpointIdx{
					Index: 1,
				})
				require.NoError(t, err)
			},
			req: &types.QueryGetValidatorSetIndexByTimestampRequest{
				Timestamp: 100,
			},
			expectedIndex: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			index, err := q.GetValidatorSetIndexByTimestamp(ctx, tc.req)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, index.Index, tc.expectedIndex)
			}
		})
	}
}
