package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgCreateReporter_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateReporter
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateReporter{
				Reporter: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCreateReporter{
				Reporter: sample.AccAddress(),
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
