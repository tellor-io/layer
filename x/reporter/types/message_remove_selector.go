package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRemoveSelector{}

func NewMsgRemoveSelector(anyAddress, selector string) *MsgRemoveSelector {
	return &MsgRemoveSelector{
		AnyAddress:      anyAddress,
		SelectorAddress: selector,
	}
}

func (msg *MsgRemoveSelector) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.AnyAddress)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	// }
	// _, err = sdk.AccAddressFromBech32(msg.SelectorAddress)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	// }
	return nil
}
