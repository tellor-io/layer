package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDelegateReporter{}

func NewMsgDelegateReporter(delegator string, reporter string, amount uint64, tokenOrigin []*TokenOrigins) *MsgDelegateReporter {
	return &MsgDelegateReporter{
		Delegator:   delegator,
		Reporter:    reporter,
		Amount:      amount,
		TokenOrigin: tokenOrigin,
	}
}

func (msg *MsgDelegateReporter) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid delegator address (%s)", err)
	}
	return nil
}
