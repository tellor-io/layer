package keeper

import (
	"context"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TRBBridgeQueryType = "TRBBridge"
)

func (k Keeper) tokenBridgeDepositCheck(ctx context.Context, queryData []byte) (types.QueryMeta, error) {
	// decode query data partial
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}
	initialArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataDecodedPartial, err := initialArgs.Unpack(queryData)
	if err != nil {
		return types.QueryMeta{}, err
	}
	if len(queryDataDecodedPartial) != 2 {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data length")
	}
	// check if first arg is a string
	if reflect.TypeOf(queryDataDecodedPartial[0]).Kind() != reflect.String {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data type")
	}
	if queryDataDecodedPartial[0].(string) != TRBBridgeQueryType {
		return types.QueryMeta{}, types.ErrNotTokenDeposit
	}
	// decode query data arguments
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return types.QueryMeta{}, err
	}

	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}

	queryDataArgsDecoded, err := queryDataArgs.Unpack(queryDataDecodedPartial[1].([]byte))
	if err != nil {
		return types.QueryMeta{}, err
	}

	if len(queryDataArgsDecoded) != 2 {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments length")
	}

	// check if first arg is a bool
	if reflect.TypeOf(queryDataArgsDecoded[0]).Kind() != reflect.Bool {
		return types.QueryMeta{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid query data arguments type")
	}

	if !queryDataArgsDecoded[0].(bool) {
		return types.QueryMeta{}, types.ErrNotTokenDeposit
	}

	queryId := utils.QueryIDFromData(queryData)
	nextId, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return types.QueryMeta{}, err
	}
	// get block timestamp
	query := types.QueryMeta{
		Id:                    nextId,
		RegistrySpecTimeframe: time.Hour,
		Expiration:            sdk.UnwrapSDKContext(ctx).BlockTime().Add(time.Hour),
		QueryType:             TRBBridgeQueryType,
		QueryId:               queryId,
		Amount:                math.NewInt(0),
	}

	return query, nil
}
