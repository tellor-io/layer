package keeper

import (
	"bytes"
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) SubmitValue(ctx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	err = k.keeper.preventBridgeWithdrawalReport(msg.QueryData)
	if err != nil {
		return nil, err
	}

	// get reporter
	reporterStake, err := k.keeper.reporterKeeper.Reporter(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}
	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporterStake, params.MinStakeAmount)
	}

	votingPower := reporterStake.Quo(layertypes.PowerReduction).Int64()
	// decode query data hex string to bytes

	queryId := utils.QueryIDFromData(msg.QueryData)

	query, err := k.keeper.Query.Get(ctx, queryId)
	if err != nil {
		// if entered here it means that there is no tip because in cycle query are initialized in genesis
		return nil, err
	}
	var incycle bool
	// get commit by identifier
	commit, err := k.keeper.Commits.Get(ctx, collections.Join(reporterAddr.Bytes(), query.Id))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		} else {
			// if there is no commit check if in cycle
			cycleQuery, err := k.keeper.GetCurrentQueryInCycleList(ctx)
			if err != nil {
				return nil, err
			}
			incycle = bytes.Equal(msg.QueryData, cycleQuery)
			err = k.keeper.directReveal(ctx, query, msg.QueryData, msg.Value, reporterAddr, votingPower, incycle)
			if err != nil {
				return nil, err
			}
			return &types.MsgSubmitValueResponse{}, nil
		}
	}

	// if there is a commit then check if its expired and verify commit, and add in cycle from commit.incycle
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if query.Expiration.Add(offset).Before(sdkCtx.BlockTime()) {
		return nil, errors.New("missed commit reveal window")
	}
	genHash := oracleutils.CalculateCommitment(msg.Value, msg.Salt)
	if genHash != commit.Hash {
		return nil, errors.New("submitted value doesn't match commitment, are you a cheater?")
	}
	incycle = commit.Incycle

	err = k.keeper.setValue(ctx, reporterAddr, query, msg.Value, msg.QueryData, votingPower, incycle)
	if err != nil {
		return nil, err
	}
	// todo: do we need to keep all the commits? also whats best do it here or aggregation?
	// remove commit from store
	// err = k.Keeper.Commits.Remove(ctx, collections.Join(reporterAddr.Bytes(), query.Id))
	// if err != nil {
	// 	return nil, err
	// }
	return &types.MsgSubmitValueResponse{}, nil
}

func (k Keeper) directReveal(ctx context.Context,
	query types.QueryMeta,
	qDataBytes []byte,
	value string,
	reporterAddr sdk.AccAddress,
	votingPower int64,
	incycle bool,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()

	if query.Amount.IsZero() && query.Expiration.Add(offset).Before(blockTime) && !incycle {
		return types.ErrNoTipsNotInCycle
	}

	if query.Amount.IsZero() && query.Expiration.Add(offset).Before(blockTime) && incycle {
		nextId, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return err
		}
		query.Id = nextId
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
	}

	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Add(offset).Before(blockTime) && !incycle {
		return errors.New("tip submission window expired and query is not in cycle")
	}

	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Add(offset).Before(blockTime) && incycle {
		query.Expiration = blockTime.Add(query.RegistrySpecTimeframe)
	}
	if query.Amount.IsZero() && blockTime.Before(query.Expiration.Add(offset)) {
		incycle = true
	}
	return k.setValue(ctx, reporterAddr, query, value, qDataBytes, votingPower, incycle)
}
