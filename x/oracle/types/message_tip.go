package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgTip = "tip"

var _ sdk.Msg = &MsgTip{}

func NewMsgTip(tipper string, queryData string, amount sdk.Coin) *MsgTip {
	return &MsgTip{
		Tipper:    tipper,
		QueryData: queryData,
		Amount:    amount,
	}
}

func (msg *MsgTip) Route() string {
	return RouterKey
}

func (msg *MsgTip) Type() string {
	return TypeMsgTip
}

func (msg *MsgTip) GetSigners() []sdk.AccAddress {
	tipper, err := sdk.AccAddressFromBech32(msg.Tipper)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{tipper}
}

func (msg *MsgTip) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgTip) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Tipper)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid tipper address (%s)", err)
	}
	return nil
}
