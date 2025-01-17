package types

import (
	layer "github.com/tellor-io/layer/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	// ensure that the fee matches the layer.BondDenom and the amount is a positive number
	if msg.Fee.Denom != layer.BondDenom || msg.Fee.Amount.IsZero() || msg.Fee.Amount.IsNegative() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid tip amount (%s)", msg.Fee.Amount.String())
	}
	if msg.Report == nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "report should not be nil")
	}
	if msg.DisputeCategory != Warning && msg.DisputeCategory != Minor && msg.DisputeCategory != Major {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "dispute category should be either Warning, Minor, or Major")
	}

	return nil
}
