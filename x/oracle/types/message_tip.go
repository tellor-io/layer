package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/tellor-io/layer/utils"
)

const TypeMsgTip = "tip"

var _ sdk.Msg = &MsgTip{}

func NewMsgTip(tipper string, queryData string, amount sdk.Coin) *MsgTip {
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
	_, err := sdk.AccAddressFromBech32(msg.Tipper)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid tipper address (%s)", err)
	}
	return nil
}
