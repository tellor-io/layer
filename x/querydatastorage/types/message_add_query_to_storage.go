package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddQueryToStorage = "add_query_to_storage"

var _ sdk.Msg = &MsgAddQueryToStorage{}

func NewMsgAddQueryToStorage(creator string, queryType string, dataTypes []string, dataFields []string) *MsgAddQueryToStorage {
	return &MsgAddQueryToStorage{
		Creator:    creator,
		QueryType:  queryType,
		DataTypes:  dataTypes,
		DataFields: dataFields,
	}
}

func (msg *MsgAddQueryToStorage) Route() string {
	return RouterKey
}

func (msg *MsgAddQueryToStorage) Type() string {
	return TypeMsgAddQueryToStorage
}

func (msg *MsgAddQueryToStorage) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgAddQueryToStorage) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddQueryToStorage) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
