package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddFeeToDispute = "add_fee_to_dispute"

var _ sdk.Msg = &MsgAddFeeToDispute{}

func NewMsgAddFeeToDispute(creator string, disputeId uint64, amount sdk.Coin, payFromBond bool) *MsgAddFeeToDispute {
	return &MsgAddFeeToDispute{
		Creator:     creator,
		DisputeId:   disputeId,
		Amount:      amount,
		PayFromBond: payFromBond,
	}
}

func (msg *MsgAddFeeToDispute) Route() string {
	return RouterKey
}

func (msg *MsgAddFeeToDispute) Type() string {
	return TypeMsgAddFeeToDispute
}

func (msg *MsgAddFeeToDispute) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddFeeToDispute) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddFeeToDispute) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
