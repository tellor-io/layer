package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgRequestAttestations = "request_attestations"

var _ sdk.Msg = &MsgRequestAttestations{}

func NewMsgRequestAttestations(creator, queryId, timestamp string) *MsgRequestAttestations {
	return &MsgRequestAttestations{
		Creator:   creator,
		QueryId:   queryId,
		Timestamp: timestamp,
	}
}

func (msg *MsgRequestAttestations) Route() string {
	return RouterKey
}

func (msg *MsgRequestAttestations) Type() string {
	return TypeMsgRequestAttestations
}

func (msg *MsgRequestAttestations) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRequestAttestations) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRequestAttestations) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.Creator)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	// }
	return nil
}
