package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRegisterQuery = "register_query"

var _ sdk.Msg = &MsgRegisterQuery{}

func NewMsgRegisterQuery(creator string, queryType string, dataTypes []string, dataFields []string) *MsgRegisterQuery {
	return &MsgRegisterQuery{
		Creator:    creator,
		QueryType:  queryType,
		DataTypes:  dataTypes,
		DataFields: dataFields,
	}
}

func (msg *MsgRegisterQuery) Route() string {
	return RouterKey
}

func (msg *MsgRegisterQuery) Type() string {
	return TypeMsgRegisterQuery
}

func (msg *MsgRegisterQuery) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRegisterQuery) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRegisterQuery) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
