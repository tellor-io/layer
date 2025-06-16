package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/math"
)

func TestMsgUpdateParams(t *testing.T) {
	tests := []struct {
		name    string
		input   *types.MsgUpdateParams
		expPass bool
		expErr  string
	}{
		{
			name: "valid params update",
			input: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params:    types.DefaultParams(),
			},
			expPass: true,
		},
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    types.DefaultParams(),
			},
			expPass: false,
			expErr:  "invalid authority address",
		},
		{
			name: "send empty params",
			input: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					AttestSlashPercentage: math.LegacyZeroDec(),
					AttestRateLimitWindow: 0,
					ValsetSlashPercentage: math.LegacyZeroDec(),
					ValsetRateLimitWindow: 0,
				},
			},
			expPass: false,
			expErr:  "attest rate limit window too small",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErr)
			}
		})
	}
}
