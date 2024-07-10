package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgWithdrawTokens_NewMsgWithdrawTokens(t *testing.T) {
	creator := sample.AccAddress()
	recipient := sample.AccAddress()
	amount := sdk.Coin{
		Denom:  "loya",
		Amount: math.NewInt(100 * 1e6),
	}
	msg := NewMsgWithdrawTokens(creator, recipient, amount)
	require.Equal(t, msg.Type(), "withdraw_tokens")
	require.Equal(t, msg.Creator, creator)
	require.Equal(t, msg.Recipient, recipient)
	require.Equal(t, msg.Amount, amount)
}

func TestMsgWithdrawTokens_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgWithdrawTokens
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgWithdrawTokens{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid address",
			msg: MsgWithdrawTokens{
				Creator: sample.AccAddress(),
			},
		},
		{
			name: "normal data",
			msg: MsgWithdrawTokens{
				Creator:   sample.AccAddress(),
				Recipient: sample.AccAddress(),
				Amount: sdk.Coin{
					Denom:  "loya",
					Amount: math.NewInt(100 * 1e6),
				},
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
