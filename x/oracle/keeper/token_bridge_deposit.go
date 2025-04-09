package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TRBBridgeQueryType = "TRBBridge"
)

// Generates a new QueryMeta for a TRBBridgeQueryType
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
	err := k.AddFirstDepositReportToQueue(ctx, query)
	if err != nil {
		return err
	}
	return k.SetValue(ctx, reporterAcc, query, value, querydata, voterPower, true)
}

func (k Keeper) AddFirstDepositReportToQueue(ctx context.Context, query types.QueryMeta) error {
	// check if deposit queue exists for this query.Id
	_, err := k.ClaimDepositQueue.Get(ctx, query.Id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		} else {
			// create new deposit queue
			depositQueue := types.DepositQueue{
				MetaId:    query.Id,
				Querydata: hex.EncodeToString(query.QueryData),
			}
			err = k.ClaimDepositQueue.Set(ctx, query.Id, depositQueue)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) AutoClaimDeposit(ctx context.Context, query types.QueryMeta) error {
	// check if deposit queue exists for this query.Id
	deposit, err := k.ClaimDepositQueue.Get(ctx, query.Id)
	if err != nil {
		return err
	}
	fmt.Println("deposit", deposit)
	return nil
}
