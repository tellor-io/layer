package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgChangeReporter{}

func NewMsgChangeReporter(delegator, reporter string) *MsgChangeReporter {
	return &MsgChangeReporter{
		DelegatorAddress: delegator,
		ReporterAddress:  reporter,
	}
}

func (msg *MsgChangeReporter) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid delegator address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	return nil
}
