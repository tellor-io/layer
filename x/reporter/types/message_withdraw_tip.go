package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgWithdrawTip{}

func NewMsgWithdrawTip(validatorAddress, selectorAddress string) *MsgWithdrawTip {
	return &MsgWithdrawTip{
		ValidatorAddress: validatorAddress,
		SelectorAddress:  selectorAddress,
	}
}

func (msg *MsgWithdrawTip) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	// }
	return nil
}
