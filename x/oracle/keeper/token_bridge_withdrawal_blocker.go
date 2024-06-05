package keeper

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k Keeper) preventBridgeWithdrawalReport(queryData []byte) error {
	// decode query data partial
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return err
	}
	initialArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataDecodedPartial, err := initialArgs.Unpack(queryData)
	if err != nil {
		return types.ErrInvalidQueryData.Wrapf("failed to unpack query data: %v", err)
	}
	if len(queryDataDecodedPartial) != 2 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data length")
	}
	// check if first arg is a string
	if reflect.TypeOf(queryDataDecodedPartial[0]).Kind() != reflect.String {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data type")
	}
	if queryDataDecodedPartial[0].(string) != TRBBridgeQueryType {
		return nil
	}
	// decode query data arguments
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return err
	}

	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}

	queryDataArgsDecoded, err := queryDataArgs.Unpack(queryDataDecodedPartial[1].([]byte))
	if err != nil {
		return types.ErrInvalidQueryData.Wrapf("failed to unpack query data arguments: %v", err)
	}

	if len(queryDataArgsDecoded) != 2 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments length")
	}

	// check if first arg is a bool
	if reflect.TypeOf(queryDataArgsDecoded[0]).Kind() != reflect.Bool {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments type")
	}

	if !queryDataArgsDecoded[0].(bool) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid report type, cannot report token bridge withdrawal")
	}
	return nil
}
