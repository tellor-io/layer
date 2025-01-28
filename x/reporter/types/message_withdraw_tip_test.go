package types

import (
	"testing"
)

func TestMsgWithdrawTip_ValidateBasic(t *testing.T) {
	// tests := []struct {
	// 	name string
	// 	msg  MsgWithdrawTip
	// 	err  error
	// }{
	// 	{
	// 		name: "invalid address",
	// 		msg: MsgWithdrawTip{
	// 			SelectorAddress: "invalid_address",
	// 		},
	// 		err: sdkerrors.ErrInvalidAddress,
	// 	}, {
	// 		name: "valid address",
	// 		msg: MsgWithdrawTip{
	// 			SelectorAddress: sample.AccAddress(),
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
