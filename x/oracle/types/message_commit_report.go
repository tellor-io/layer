package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgCommitReport = "commit_report"

var _ sdk.Msg = &MsgCommitReport{}

func NewMsgCommitReport(creator string, queryData string, hash string) *MsgCommitReport {
	return &MsgCommitReport{
		Creator:   creator,
		QueryData: queryData,
		Hash:      hash,
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

func (msg *MsgCommitReport) GetSignerAndValidateMsg() (sdk.AccAddress, error) {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.QueryData == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "query data field cannot be empty")
	}
	if msg.Hash == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "hash field cannot be empty")
	}
	return addr, nil
}
