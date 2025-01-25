package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgSelectReporter{}

func NewMsgSelectReporter(selector, reporter string) *MsgSelectReporter {
	return &MsgSelectReporter{
		SelectorAddress: selector,
		ReporterAddress: reporter,
	}
}

func (msg *MsgSelectReporter) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	// }
	// _, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	// }
	return nil
}
