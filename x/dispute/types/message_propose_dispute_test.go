package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgProposeDispute_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgProposeDispute
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgProposeDispute{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid address, bad coins",
			msg: MsgProposeDispute{
				Creator: sample.AccAddress(),
				Fee:     sdk.NewCoin("badcoin", math.NewInt(1000000)),
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid address, valid coins, nil report",
			msg: MsgProposeDispute{
				Creator: sample.AccAddress(),
				Fee:     sdk.NewCoin(layer.BondDenom, math.NewInt(1000000)),
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid address, valid coins, valid report, valid category",
			msg: MsgProposeDispute{
				Creator:         sample.AccAddress(),
				Fee:             sdk.NewCoin(layer.BondDenom, math.NewInt(1000000)),
				Report:          &oracletypes.MicroReport{},
				DisputeCategory: Warning,
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
