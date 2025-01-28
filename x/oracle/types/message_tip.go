package types

import (
	"github.com/tellor-io/layer/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgTip = "tip"

var _ sdk.Msg = &MsgTip{}

func NewMsgTip(tipper, queryData string, amount sdk.Coin) *MsgTip {
	queryDataBz, err := utils.QueryBytesFromString(queryData)
	if err != nil {
		panic(err)
	}

	return &MsgTip{
		Tipper:    tipper,
		QueryData: queryDataBz,
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
	// _, err := sdk.AccAddressFromBech32(msg.Tipper)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid tipper address (%s)", err)
	// }
	// // ensure that the msg.Amount.Denom matches the layer.BondDenom and the amount is a positive number
	// if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid tip amount (%s)", msg.Amount.String())
	// }
	// // ensure that the queryData is not empty
	// if len(msg.QueryData) == 0 {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query data is empty")
	// }
	return nil
}
