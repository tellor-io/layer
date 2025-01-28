package types

import (
	"testing"
)

func TestMsgCreateReporter_ValidateBasic(t *testing.T) {
	// tests := []struct {
	// 	name string
	// 	msg  MsgCreateReporter
	// 	err  error
	// }{
	// 	{
	// 		name: "invalid address",
	// 		msg: MsgCreateReporter{
	// 			ReporterAddress:   "invalid_address",
	// 			CommissionRate:    math.LegacyNewDec(1),
	// 			MinTokensRequired: math.NewInt(1000000),
	// 		},
	// 		err: sdkerrors.ErrInvalidAddress,
	// 	}, {
	// 		name: "valid address",
	// 		msg: MsgCreateReporter{
	// 			ReporterAddress:   sample.AccAddress(),
	// 			CommissionRate:    math.LegacyNewDec(1),
	// 			MinTokensRequired: math.NewInt(1000000),
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
