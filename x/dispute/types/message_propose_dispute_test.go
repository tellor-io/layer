package types

import (
	"testing"
)

func TestMsgProposeDispute_ValidateBasic(t *testing.T) {
	// tests := []struct {
	// 	name string
	// 	msg  MsgProposeDispute
	// 	err  error
	// }{
	// 	{
	// 		name: "invalid address",
	// 		msg: MsgProposeDispute{
	// 			Creator: "invalid_address",
	// 		},
	// 		err: sdkerrors.ErrInvalidAddress,
	// 	},
	// 	{
	// 		name: "valid address, bad coins",
	// 		msg: MsgProposeDispute{
	// 			Creator: sample.AccAddress(),
	// 			Fee:     sdk.NewCoin("badcoin", math.NewInt(1000000)),
	// 		},
	// 		err: sdkerrors.ErrInvalidCoins,
	// 	},
	// 	{
	// 		name: "valid address, valid coins, nil report",
	// 		msg: MsgProposeDispute{
	// 			Creator: sample.AccAddress(),
	// 			Fee:     sdk.NewCoin(layer.BondDenom, math.NewInt(1000000)),
	// 		},
	// 		err: sdkerrors.ErrInvalidRequest,
	// 	},
	// 	{
	// 		name: "valid address, valid coins, valid report, valid category",
	// 		msg: MsgProposeDispute{
	// 			Creator:         sample.AccAddress(),
	// 			Fee:             sdk.NewCoin(layer.BondDenom, math.NewInt(1000000)),
	// 			Report:          &oracletypes.MicroReport{},
	// 			DisputeCategory: Warning,
	// 		},
	// 		err: nil,
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
