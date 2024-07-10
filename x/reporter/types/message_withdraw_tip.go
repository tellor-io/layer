package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgWithdrawTip{}

func NewMsgWithdrawTip(validatorAddress, selectorAddress string) *MsgWithdrawTip {
	return &MsgWithdrawTip{
		ValidatorAddress: validatorAddress,
		SelectorAddress:  selectorAddress,
	}
}

func (msg *MsgWithdrawTip) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
