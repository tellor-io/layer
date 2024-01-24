package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCommitReport = "commit_report"

var _ sdk.Msg = &MsgCommitReport{}

func NewMsgCommitReport(creator string, queryData string, value string) *MsgCommitReport {
	return &MsgCommitReport{
		Creator:     creator,
		QueryData:   queryData,
		SaltedValue: value,
	}
}

func (msg *MsgCommitReport) Route() string {
	return RouterKey
}

func (msg *MsgCommitReport) Type() string {
	return TypeMsgCommitReport
}

func (msg *MsgCommitReport) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCommitReport) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCommitReport) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
