package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgRegisterSpec_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRegisterSpec
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgRegisterSpec{
				Registrar: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgRegisterSpec{
				Registrar: sample.AccAddress(),
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

func TestMsgRegisterSpec_NewMsgRegisterSpec(t *testing.T) {
	require := require.New(t)

	registrar := sample.AccAddress()
	queryType := "SpotPrice"
	abiComponents := []*ABIComponent{
		{Name: "asset", FieldType: "string"},
		{Name: "currency", FieldType: "string"},
	}
	msg := NewMsgRegisterSpec(registrar, queryType, &DataSpec{
		DocumentHash:      "document_hash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         registrar,
		ReportBlockWindow: 10,
		AbiComponents:     abiComponents,
	})
	require.Equal(msg.Spec.AbiComponents, abiComponents)
	require.Equal(msg.Spec.DocumentHash, "document_hash")
	require.Equal(msg.Spec.ResponseValueType, "uint256")
	require.Equal(msg.Spec.AggregationMethod, "weighted-median")
	require.Equal(msg.Spec.Registrar, registrar)
	require.Equal(msg.Spec.ReportBlockWindow, uint64(10))
	require.Equal(msg.Registrar, registrar)
	require.Equal(msg.QueryType, queryType)
}
