package types

import (
	"testing"
)

func TestMsgSelectReporter_ValidateBasic(t *testing.T) {
	// tests := []struct {
	// 	name string
	// 	msg  MsgSelectReporter
	// 	err  error
	// }{
	// 	{
	// 		name: "invalid address",
	// 		msg: MsgSelectReporter{
	// 			SelectorAddress: "invalid_address",
	// 		},
	// 		err: sdkerrors.ErrInvalidAddress,
	// 	}, {
	// 		name: "valid address",
	// 		msg: MsgSelectReporter{
	// 			SelectorAddress: sample.AccAddress(),
	// 			ReporterAddress: sample.AccAddress(),
	// 		},
	// 	},
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		err := tt.msg.ValidateBasic()
	// 		if tt.err != nil {
	// 			require.ErrorIs(t, err, tt.err)
	// 			return
	// 		}
	// 		require.NoError(t, err)
	// 	})
	// }
}
