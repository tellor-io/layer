package types

import (
	"testing"
)

func TestMsgVote_ValidateBasic(t *testing.T) {
	// tests := []struct {
	// 	name string
	// 	msg  MsgVote
	// 	err  error
	// }{
	// 	{
	// 		name: "invalid address",
	// 		msg: MsgVote{
	// 			Voter: "invalid_address",
	// 		},
	// 		err: sdkerrors.ErrInvalidAddress,
	// 	}, {
	// 		name: "valid address",
	// 		msg: MsgVote{
	// 			Voter: sample.AccAddress(),
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
