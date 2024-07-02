package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgRequestAttestations_NewMsgRequestAttestations(t *testing.T) {
	addr := sample.AccAddress()
	qId := "queryId"
	timestamp := "1234567890"
	msg := NewMsgRequestAttestations(addr, qId, timestamp)
	require.Equal(t, msg.Type(), "request_attestations")
	require.Equal(t, msg.Creator, addr)
	require.Equal(t, msg.QueryId, qId)
	require.Equal(t, msg.Timestamp, timestamp)
}

func TestMsgRequestAttestations_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRequestAttestations
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgRequestAttestations{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid address",
			msg: MsgRequestAttestations{
				Creator: sample.AccAddress(),
			},
		},
		{
			name: "normal data",
			msg: MsgRequestAttestations{
				Creator:   sample.AccAddress(),
				QueryId:   "queryId",
				Timestamp: "1234567890",
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
