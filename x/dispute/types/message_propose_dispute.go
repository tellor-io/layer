package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

const TypeMsgProposeDispute = "propose_dispute"

var _ sdk.Msg = &MsgProposeDispute{}

func NewMsgProposeDispute(creator string, report *oracletypes.MicroReport, disputeCategory DisputeCategory, fee sdk.Coin, payFromBond bool) *MsgProposeDispute {
	return &MsgProposeDispute{
		Creator:         creator,
		Report:          report,
		DisputeCategory: disputeCategory,
		Fee:             fee,
		PayFromBond:     payFromBond,
	}
}

func (msg *MsgProposeDispute) Route() string {
	return RouterKey
}

func (msg *MsgProposeDispute) Type() string {
	return TypeMsgProposeDispute
}

func (msg *MsgProposeDispute) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgProposeDispute) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgProposeDispute) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
