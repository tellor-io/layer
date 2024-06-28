package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgClaimDeposit_NewMsgClaimDepositRequest(t *testing.T) {
	addr := sample.AccAddress()
	msg := NewMsgClaimDepositRequest(addr, 0, 0)
	require.Equal(t, msg.Type(), "claim_deposit")
	require.Equal(t, msg.Creator, addr)
	require.Equal(t, msg.DepositId, uint64(0))
	require.Equal(t, msg.Index, uint64(0))
}

func TestMsgClaimDeposit_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgClaimDepositRequest
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgClaimDepositRequest{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgClaimDepositRequest{
				Creator: sample.AccAddress(),
			},
		},
		{
			name: "normal data",
			msg: MsgClaimDepositRequest{
				Creator:   sample.AccAddress(),
				DepositId: 1,
				Index:     1,
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