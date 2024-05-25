package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUndelegateReporter{}

func NewMsgUndelegateReporter(delegator string, tokenOrigin []*TokenOrigin) *MsgUndelegateReporter {
	return &MsgUndelegateReporter{
		DelegatorAddress: delegator,
		TokenOrigins:     tokenOrigin,
	}
}

func (msg *MsgUndelegateReporter) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid delegator address (%s)", err)
	}
	return nil
}
