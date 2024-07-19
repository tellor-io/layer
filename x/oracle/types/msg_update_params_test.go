package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	require := require.New(t)

	// empty authority
	msg := MsgUpdateParams{}
	require.ErrorContains(msg.ValidateBasic(), "invalid authority address")

	// bad authority address
	msg = MsgUpdateParams{
		Authority: "bad_address",
	}
	require.ErrorContains(msg.ValidateBasic(), "invalid authority address")

	// good authority, anything geos for params ?
	msg = MsgUpdateParams{
		Authority: sample.AccAddress(),
		Params:    Params{},
	}
	require.NoError(msg.ValidateBasic())
}

func TestMsgUpdateParams_TestGetSigners(t *testing.T) {
	require := require.New(t)

	// bad signer address
	msg := MsgUpdateParams{
		Authority: "bad_address",
	}
	require.Panics(func() {
		msg.GetSigners()
	})

	// good signer address
	signer := sdk.AccAddress(sample.AccAddress())
	msg = MsgUpdateParams{
		Authority: signer.String(),
	}
	require.Equal([]sdk.AccAddress{signer}, msg.GetSigners())
}

func TestMsgUpdateParams_GetSignBytes(t *testing.T) {
	require := require.New(t)

	msg := MsgUpdateParams{
		Authority: sample.AccAddress(),
	}
	msgBz := ModuleCdc.MustMarshalJSON(&msg)
	require.Equal(msgBz, msg.GetSignBytes())
}