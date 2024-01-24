package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRegisterSpec = "register_spec"

var _ sdk.Msg = &MsgRegisterSpec{}

func NewMsgRegisterSpec(creator string, queryType string, spec *DataSpec) *MsgRegisterSpec {
	return &MsgRegisterSpec{
		Creator:   creator,
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
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRegisterSpec) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRegisterSpec) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
