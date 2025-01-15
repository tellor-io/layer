package keeper

import (
	"context"
	"errors"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// PreventBridgeWithdrawalReport verifies if the queryData is a TRBBridgeQueryType. If not, it returns false, nil.
// If it is, then it further checks whether it is a withdrawal or deposit report. If it is a withdrawal report, it returns an error
// indicating that such reports should not be processed.
func (k Keeper) PreventBridgeWithdrawalReport(ctx context.Context, queryData []byte) (bool, error) {
	// decode query data partial
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return false, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return false, err
	}
	initialArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataDecodedPartial, err := initialArgs.Unpack(queryData)
	if err != nil {
		return false, types.ErrInvalidQueryData.Wrapf("failed to unpack query data: %v", err)
	}
	if len(queryDataDecodedPartial) != 2 {
		return false, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data length")
	}
	// check if first arg is a string
	if reflect.TypeOf(queryDataDecodedPartial[0]).Kind() != reflect.String {
		return false, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data type")
	}
	if queryDataDecodedPartial[0].(string) != TRBBridgeQueryType {
		return false, nil
	}
	// decode query data arguments
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return false, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return false, err
	}

	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}

	queryDataArgsDecoded, err := queryDataArgs.Unpack(queryDataDecodedPartial[1].([]byte))
	if err != nil {
		return false, types.ErrInvalidQueryData.Wrapf("failed to unpack query data arguments: %v", err)
	}

	if len(queryDataArgsDecoded) != 2 {
		return false, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments length")
	}

	// check if first arg is a bool
	if reflect.TypeOf(queryDataArgsDecoded[0]).Kind() != reflect.Bool {
		return false, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments type")
	}

	if !queryDataArgsDecoded[0].(bool) {
		return false, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid report type, cannot report token bridge withdrawal")
	}
	// get depositId status
	depositId := queryDataArgsDecoded[1].(*big.Int).Uint64()
	claimStatus, err := k.bridgeKeeper.GetDepositStatus(ctx, depositId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return true, nil
		}
		return false, err
	}
	if claimStatus {
		return false, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid report type, cannot report token bridge deposit that has already been claimed")
	}
	return true, nil
}
