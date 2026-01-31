package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TRBBridgeQueryType   = "TRBBridge"
	TRBBridgeV2QueryType = "TRBBridgeV2"
)

// Generates a new QueryMeta for a TRBBridgeV2QueryType
func (k Keeper) TokenBridgeDepositQuery(ctx context.Context, queryData []byte) (types.QueryMeta, error) {
	// decode query data partial
	nextId, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return types.QueryMeta{}, err
	}
	// get block timestamp
	query := types.QueryMeta{
		Id:                      nextId,
		RegistrySpecBlockWindow: 2000,
		Expiration:              uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) + 2000,
		QueryType:               TRBBridgeV2QueryType,
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
