package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgSubmitOracleAttestation = "submit_oracle_attestation"

var _ sdk.Msg = &MsgSubmitOracleAttestation{}

func NewMsgSubmitOracleAttestation(creator string, queryId string, timestamp string, signature string) *MsgSubmitOracleAttestation {
	return &MsgSubmitOracleAttestation{
		Creator:   creator,
		QueryId:   queryId,
		Timestamp: timestamp,
		Signature: signature,
	}
}

func (msg *MsgSubmitOracleAttestation) Route() string {
	return RouterKey
}

func (msg *MsgSubmitOracleAttestation) Type() string {
	return TypeMsgSubmitBridgeValsetSignature
}

func (msg *MsgSubmitOracleAttestation) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSubmitOracleAttestation) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubmitOracleAttestation) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
