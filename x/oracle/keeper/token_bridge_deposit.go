package keeper

import (
	"context"
	"encoding/hex"
	"reflect"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
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
		Id:                    nextId,
		RegistrySpecTimeframe: time.Hour,
		Expiration:            sdk.UnwrapSDKContext(ctx).BlockTime().Add(time.Hour),
		QueryType:             TRBBridgeQueryType,
		QueryData:             queryData,
		Amount:                math.NewInt(0),
		CycleList:             true,
	}

	return query, nil
}

func (k Keeper) HandleBridgeDepositCommit(ctx context.Context, queryId []byte, query types.QueryMeta, reporterAcc sdk.AccAddress, hash string) error {
	sdkctx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkctx.BlockTime()

	if query.Amount.IsZero() && query.Expiration.Before(blockTime) {

		nextId, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return err
		}
		query.Id = nextId
		// reset query fields when generating next id
		query.HasRevealedReports = false
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
		err = k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		if err != nil {
			return err
		}
	}

	offset, err := k.GetReportOffsetParam(ctx)
	if err != nil {
		return err
	}

	// if there is tip but window expired, only bridgeDeposit(bd) query can extend the window when its a bd query, otherwise requires tip vi msgTip tx
	// if tip amount is greater than zero and query timeframe plus offset is expired, it means that the query didn't have any revealed reports
	// and the tip is still there and so the time can be extended only if the query is a bridge deposit or via a tip transaction
	// maintains the same id until the query is paid out
	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Add(offset).Before(blockTime) {
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
		err := k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		if err != nil {
			return err
		}
	}

	if query.Expiration.Before(blockTime) {
		return types.ErrCommitWindowExpired.Wrapf("query for bridge deposit is expired")
	}

	commit := types.Commit{
		Reporter: reporterAcc.String(),
		QueryId:  queryId,
		Hash:     hash,
		Incycle:  query.CycleList,
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_commit",
			sdk.NewAttribute("reporter", reporterAcc.String()),
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("commit_id", strconv.FormatUint(query.Id, 10)),
		),
	})
	return k.Commits.Set(ctx, collections.Join(reporterAcc.Bytes(), query.Id), commit)
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
	blockTime := sdkCtx.BlockTime()

	offset, err := k.GetReportOffsetParam(ctx)
	if err != nil {
		return err
	}

	if query.Amount.IsZero() && query.Expiration.Add(offset).Before(blockTime) {
		nextId, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return err
		}
		query.Id = nextId
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
	}
	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Add(offset).Before(blockTime) {
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
	}

	if query.Expiration.Before(blockTime) {
		return types.ErrSubmissionWindowExpired.Wrapf("query for bridge deposit is expired")
	}
	return k.SetValue(ctx, reporterAcc, query, value, querydata, voterPower, true)
}
