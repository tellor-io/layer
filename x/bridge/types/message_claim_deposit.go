package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgClaimDeposit = "claim_deposit"

var _ sdk.Msg = &MsgClaimDepositRequest{}

func NewMsgClaimDepositRequest(creator string, depositId uint64, reportIndex uint64) *MsgClaimDepositRequest {
	return &MsgClaimDepositRequest{
		Creator:   creator,
		DepositId: depositId,
		Index:     reportIndex,
	}
}

func (msg *MsgClaimDepositRequest) Route() string {
	return RouterKey
}

func (msg *MsgClaimDepositRequest) Type() string {
	return TypeMsgClaimDeposit
}

func (msg *MsgClaimDepositRequest) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgClaimDepositRequest) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgClaimDepositRequest) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
