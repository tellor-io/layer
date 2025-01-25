package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgWithdrawTokens = "withdraw_tokens"

var _ sdk.Msg = &MsgWithdrawTokens{}

func NewMsgWithdrawTokens(creator, recipient string, amount sdk.Coin) *MsgWithdrawTokens {
	return &MsgWithdrawTokens{
		Creator:   creator,
		Recipient: recipient,
		Amount:    amount,
	}
}

func (msg *MsgWithdrawTokens) Route() string {
	return RouterKey
}

func (msg *MsgWithdrawTokens) Type() string {
	return TypeMsgWithdrawTokens
}

func (msg *MsgWithdrawTokens) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgWithdrawTokens) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgWithdrawTokens) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.Creator)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	// }
	return nil
}
