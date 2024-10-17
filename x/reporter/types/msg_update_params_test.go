package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	"cosmossdk.io/math"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	require := require.New(t)

	// invalid authority
	msg := MsgUpdateParams{
		Authority: "invalid_address",
	}
	require.ErrorContains(msg.ValidateBasic(), "invalid authority address")

	// valid authority, no params
	msg = MsgUpdateParams{
		Authority: sample.AccAddress(),
	}
	require.NoError(msg.ValidateBasic())

	// valid authority, valid params
	msg = MsgUpdateParams{
		Authority: sample.AccAddress(),
		Params: Params{
			MinCommissionRate: math.NewUint(5),
			MinTrb:            math.NewInt(1),
			MaxSelectors:      100,
		},
	}
	require.NoError(msg.ValidateBasic())
}
