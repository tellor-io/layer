package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgSubmitValue_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSubmitValue
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSubmitValue{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid address, empty query data",
			msg: MsgSubmitValue{
				Creator:   sample.AccAddress(),
				QueryData: []byte(""),
			},
			err: errors.New("MsgSubmitValue query data cannot be empty (%s)"),
		},
		{
			name: "valid address, nonempty queryData, empty value",
			msg: MsgSubmitValue{
				Creator:   sample.AccAddress(),
				QueryData: []byte("query_data"),
				Value:     "",
			},
			err: errors.New("MsgSubmitValue value field cannot be empty (%s)"),
		},
		{
			name: "valid address, nonempty queryData, nonempty value",
			msg: MsgSubmitValue{
				Creator:   sample.AccAddress(),
				QueryData: []byte("query_data"),
				Value:     "value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorContains(t, err, tt.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgSubmitValue_GetSignerAndValidateMsg(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgSubmitValue
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSubmitValue{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid address, empty query data",
			msg: MsgSubmitValue{
				Creator: creator,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid address and query data, empty value",
			msg: MsgSubmitValue{
				Creator:   creator,
				QueryData: []byte("query_data"),
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid address, query data, and value",
			msg: MsgSubmitValue{
				Creator:   creator,
				QueryData: []byte("query_data"),
				Value:     "value",
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
