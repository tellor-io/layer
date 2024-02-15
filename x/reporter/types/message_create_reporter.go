package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var _ sdk.Msg = &MsgCreateReporter{}

func NewMsgCreateReporter(reporter string, amount math.Int, tokenOrigins []*TokenOrigin, commission *stakingtypes.Commission) *MsgCreateReporter {
	return &MsgCreateReporter{
		Reporter:     reporter,
		Amount:       amount,
		TokenOrigins: tokenOrigins,
		Commission:   commission,
	}
}

func (msg *MsgCreateReporter) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Reporter)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
