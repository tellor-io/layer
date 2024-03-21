package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	layertypes "github.com/tellor-io/layer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}
	// get reporter
	reporter, err := k.reporterKeeper.Reporter(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}
	if reporter.Jailed {
		return nil, errorsmod.Wrapf(types.ErrReporterJailed, "reporter %s is in jail", reporterAddr)
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if reporter.TotalTokens.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporter.TotalTokens, params.MinStakeAmount)
	}

	votingPower := reporter.TotalTokens.Quo(layertypes.PowerReduction).Int64()
	// decode query data hex string to bytes
	msg.QueryData = utils.Remove0xPrefix(msg.QueryData)
	qDataBytes, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data string: %v", err))
	}
	queryId := utils.QueryIDFromData(qDataBytes)

	query, err := k.Keeper.Query.Get(ctx, queryId)
	if err != nil {
		// if entered here it means that there is no tip because in cycle query are initialized in genesis
		return nil, err
	}
	var incycle bool
	// get commit by identifier
	commit, err := k.Keeper.Commits.Get(ctx, collections.Join(reporterAddr.Bytes(), query.Id))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		} else {
			// if there is no commit check if in cycle
			cycleQuery, err := k.Keeper.GetCurrentQueryInCycleList(ctx)
			if err != nil {
				return nil, err
			}
			incycle = msg.QueryData == cycleQuery
			err = k.directReveal(ctx, query, qDataBytes, msg.Value, reporterAddr, votingPower, incycle)
			if err != nil {
				return nil, err
			}
			return &types.MsgSubmitValueResponse{}, nil
		}
	}
	// // if there is a commit then check if its expired and verify commit, and add in cycle from commit.incycle
	// if query.Expiration.Add(offset).Before(ctx.BlockTime()) {
	// 	return nil, errors.New("missed commit reveal window")
	// }
	genHash := oracleutils.CalculateCommitment(msg.Value, msg.Salt)
	if genHash != commit.Hash {
		return nil, errors.New("submitted value doesn't match commitment, are you a cheater?")
	}
	incycle = commit.Incycle

	err = k.setValue(ctx, reporterAddr, query, msg.Value, qDataBytes, votingPower, incycle)
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

func (k Keeper) directReveal(ctx sdk.Context,
	query types.QueryMeta,
	qDataBytes []byte,
	value string,
	reporterAddr sdk.AccAddress,
	votingPower int64,
	incycle bool) error {

	if query.Amount.IsZero() && query.Expiration.Add(offset).Before(ctx.BlockTime()) && !incycle {
		return types.ErrNoTipsNotInCycle
	}

	if query.Amount.IsZero() && query.Expiration.Add(offset).Before(ctx.BlockTime()) && incycle {
		nextId, err := k.QuerySequnecer.Next(ctx)
		if err != nil {
			return err
		}
		query.Id = nextId
		query.Expiration = ctx.BlockTime().Add(query.RegistrySpecTimeframe)
	}

	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Add(offset).Before(ctx.BlockTime()) && !incycle {
		return errors.New("tip submission window expired and query is not in cycle")
	}

	if query.Amount.GT(math.ZeroInt()) && query.Expiration.Add(offset).Before(ctx.BlockTime()) && incycle {
		query.Expiration = ctx.BlockTime().Add(query.RegistrySpecTimeframe)
	}
	if query.Amount.IsZero() && ctx.BlockTime().Before(query.Expiration.Add(offset)) {
		incycle = true
	}
	err := k.setValue(ctx, reporterAddr, query, value, qDataBytes, votingPower, incycle)
	if err != nil {
		return err
	}

	return nil
}
