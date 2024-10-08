package keeper

import (
	"context"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TRBBridgeQueryType = "TRBBridge"
)

func (k Keeper) TokenBridgeDepositCheck(ctx context.Context, queryData []byte) (types.QueryMeta, error) {
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

	nextId, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return types.QueryMeta{}, err
	}
	// get block timestamp
	query := types.QueryMeta{
		Id:                      nextId,
		RegistrySpecBlockWindow: 2000,
		Expiration:              uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) + 2000,
		QueryType:               TRBBridgeQueryType,
		QueryData:               queryData,
		Amount:                  math.NewInt(0),
		CycleList:               true,
	}

	return query, nil
}

func (k Keeper) HandleBridgeDepositDirectReveal(
	ctx context.Context,
	query types.QueryMeta,
	querydata []byte,
	reporterAcc sdk.AccAddress,
	value string,
	voterPower uint64,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	if query.Amount.IsZero() && query.Expiration <= uint64(blockHeight) {
		nextId, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return err
		}
		query.Id = nextId
		query.Expiration = uint64(blockHeight) + query.RegistrySpecBlockWindow
	}
	if query.Amount.GT(math.ZeroInt()) && query.Expiration <= uint64(blockHeight) {
		query.Expiration = uint64(blockHeight) + query.RegistrySpecBlockWindow
	}

	if query.Expiration < uint64(blockHeight) {
		return types.ErrSubmissionWindowExpired.Wrapf("query for bridge deposit is expired")
	}
	return k.SetValue(ctx, reporterAcc, query, value, querydata, voterPower, true)
}
