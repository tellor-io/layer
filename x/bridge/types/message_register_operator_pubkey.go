package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRegisterOperatorPubkey = "register_operator_pubkey"

var _ sdk.Msg = &MsgRegisterOperatorPubkey{}

func NewMsgRegisterOperatorPubkey(creator string, operatorPubkey string) *MsgRegisterOperatorPubkey {
	return &MsgRegisterOperatorPubkey{
		Creator:        creator,
		OperatorPubkey: operatorPubkey,
	}
}

func (msg *MsgRegisterOperatorPubkey) Route() string {
	return RouterKey
}

func (msg *MsgRegisterOperatorPubkey) Type() string {
	return TypeMsgRegisterOperatorPubkey
}

func (msg *MsgRegisterOperatorPubkey) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRegisterOperatorPubkey) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRegisterOperatorPubkey) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
