package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgCommitReport_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCommitReport
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCommitReport{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCommitReport{
				Creator: sample.AccAddress(),
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

func TestMsgCommitReport_GetSignerAndValidateMsg(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgCommitReport
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCommitReport{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid address, empty query data",
			msg: MsgCommitReport{
				Creator: creator,
				Hash:    "hash",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid address and query data, empty hash",
			msg: MsgCommitReport{
				Creator:   creator,
				QueryData: []byte("query_data"),
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid address, query data, and hash",
			msg: MsgCommitReport{
				Creator:   creator,
				QueryData: []byte("query_data"),
				Hash:      "hash",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := tt.msg.GetSignerAndValidateMsg()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.msg.Creator, signer.String())
		})
	}
}