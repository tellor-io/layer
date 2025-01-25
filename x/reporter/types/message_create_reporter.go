package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCreateReporter{}

func NewMsgCreateReporter(reporter string, commission math.LegacyDec) *MsgCreateReporter {
	return &MsgCreateReporter{
		ReporterAddress: reporter,
		CommissionRate:  commission,
	}
}

func (msg *MsgCreateReporter) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	// }

	// // check that mintokensrequired is positive
	// if msg.MinTokensRequired.LTE(math.ZeroInt()) {
	// 	return errors.New("MinTokensRequired must be positive (%s)")
	// }

	return nil
}
