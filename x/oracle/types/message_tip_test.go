package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
)

func TestMsgTip_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgTip
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgTip{
				Tipper: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgTip{
				Tipper: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
