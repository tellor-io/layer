package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubmitValue = "submit_value"

var _ sdk.Msg = &MsgSubmitValue{}

func NewMsgSubmitValue(creator string, queryData string, value string, salt string) *MsgSubmitValue {
	return &MsgSubmitValue{
		Creator:   creator,
		QueryData: queryData,
		Value:     value,
		Salt:      salt,
	}
}

func (msg *MsgSubmitValue) Route() string {
	return RouterKey
}

func (msg *MsgSubmitValue) Type() string {
	return TypeMsgSubmitValue
}

func (msg *MsgSubmitValue) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubmitValue) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubmitValue) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

func (msg *MsgSubmitValue) GetSignerAndValidateMsg() (sdk.AccAddress, error) {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.QueryData == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "query data cannot be empty")
	}
	if msg.Value == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be empty")
	}
	return addr, nil
}
