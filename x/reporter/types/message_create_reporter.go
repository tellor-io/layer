package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateReporter{}

func NewMsgCreateReporter(reporter string, amount uint64, tokenOrigins []*TokenOrigins) *MsgCreateReporter {
	return &MsgCreateReporter{
		Reporter:     reporter,
		Amount:       amount,
		TokenOrigins: tokenOrigins,
	}
}

func (msg *MsgCreateReporter) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Reporter)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
