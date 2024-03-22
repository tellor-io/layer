package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubmitBridgeValsetSignature = "submit_bridge_valset_signature"

var _ sdk.Msg = &MsgSubmitBridgeValsetSignature{}

func NewMsgSubmitBridgeValsetSignature(creator string, timestamp string, signature string) *MsgSubmitBridgeValsetSignature {
	return &MsgSubmitBridgeValsetSignature{
		Creator:   creator,
		Timestamp: timestamp,
		Signature: signature,
	}
}

func (msg *MsgSubmitBridgeValsetSignature) Route() string {
	return RouterKey
}

func (msg *MsgSubmitBridgeValsetSignature) Type() string {
	return TypeMsgSubmitBridgeValsetSignature
}

func (msg *MsgSubmitBridgeValsetSignature) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubmitBridgeValsetSignature) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubmitBridgeValsetSignature) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
