package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgVote = "vote"

var _ sdk.Msg = &MsgVote{}

func NewMsgVote(voter string, id uint64, vote VoteEnum) *MsgVote {
	return &MsgVote{
		Voter: voter,
		Id:    id,
		Vote:  vote,
	}
}

func (msg *MsgVote) Route() string {
	return RouterKey
}

func (msg *MsgVote) Type() string {
	return TypeMsgVote
}

func (msg *MsgVote) GetSigners() []sdk.AccAddress {
	voter, err := sdk.AccAddressFromBech32(msg.Voter)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{voter}
}

func (msg *MsgVote) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgVote) ValidateBasic() error {
	// _, err := sdk.AccAddressFromBech32(msg.Voter)
	// if err != nil {
	// 	return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid voter address (%s)", err)
	// }
	return nil
}
