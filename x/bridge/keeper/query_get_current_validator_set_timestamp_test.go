package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestQuery_GetCurrentValidatorSetTimestamp(t *testing.T) {
	k, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	q := keeper.NewQuerier(k)

	testCases := []struct {
		name              string
		setup             func()
		req               *types.QueryGetCurrentValidatorSetTimestampRequest
		expectedTimestamp uint64
		err               bool
	}{
		{
			name: "nil request",
			req:  nil,
			err:  true,
		},
		{
			name: "LatestCheckpointIdx not set",
			req:  &types.QueryGetCurrentValidatorSetTimestampRequest{},
			err:  true,
		},
		{
			name: "success",
			setup: func() {
				err := k.LatestCheckpointIdx.Set(ctx, types.CheckpointIdx{
					Index: 1,
				})
				require.NoError(t, err)
				err = k.ValidatorCheckpointIdxMap.Set(ctx, 1, types.CheckpointTimestamp{
					Timestamp: 100,
				})
				require.NoError(t, err)
			},
			req:               &types.QueryGetCurrentValidatorSetTimestampRequest{},
			err:               false,
			expectedTimestamp: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			timestamp, err := q.GetCurrentValidatorSetTimestamp(ctx, tc.req)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, timestamp.Timestamp, tc.expectedTimestamp)
			}
		})
	}
}
