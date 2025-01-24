package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
		},
		{
			name: "valid address, empty coin",
			msg: MsgTip{
				Tipper: sample.AccAddress(),
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid address, valid coin, empty query data",
			msg: MsgTip{
				Tipper: sample.AccAddress(),
				Amount: sdk.NewCoin(layer.BondDenom, math.NewInt(1000000)),
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid address, valid coin, valid query data",
			msg: MsgTip{
				Tipper:    sample.AccAddress(),
				Amount:    sdk.NewCoin(layer.BondDenom, math.NewInt(1000000)),
				QueryData: []byte("test"),
			},
			err: nil,
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
