package types

import (
	"reflect"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRegisterSpec = "register_spec"

var _ sdk.Msg = &MsgRegisterSpec{}

func NewMsgRegisterSpec(registrar, queryType string, spec *DataSpec) *MsgRegisterSpec {
	return &MsgRegisterSpec{
		Registrar: registrar,
		QueryType: queryType,
		Spec:      *spec,
	}
}

func (msg *MsgRegisterSpec) Route() string {
	return RouterKey
}

func (msg *MsgRegisterSpec) Type() string {
	return TypeMsgRegisterSpec
}

func (msg *MsgRegisterSpec) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Registrar)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRegisterSpec) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Registrar)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// querytype should be non-empty string
	if reflect.TypeOf(msg.QueryType).Kind() != reflect.String {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query type must be a string")
	}
	if msg.QueryType == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query type must be a non-empty string")
	}
	//  ensure correctness of data within the Spec
	if msg.Spec.AbiComponents == nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec abi components should not be empty")
	}
	if msg.Spec.AggregationMethod == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec aggregation method should not be empty")
	}
	if msg.Spec.Registrar == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec registrar should not be empty")
	}
	if msg.Spec.ResponseValueType == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec response value type should not be empty")
	}

	return nil
}
