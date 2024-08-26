package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CommitReport(ctx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()
	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	// get reporter
	reporterStake, err := k.keeper.reporterKeeper.ReporterStake(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required amount is %s", reporterStake, params.MinStakeAmount)
	}

	// get query id bytes hash from query data
	queryId := utils.QueryIDFromData(msg.QueryData)

	// get query info by query id
	query, err := k.keeper.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		// check if query is token bridge deposit, cyclists should have queries
		query, err = k.keeper.TokenBridgeDepositCheck(ctx, msg.QueryData)
		if err != nil {
			if errors.Is(err, types.ErrNotTokenDeposit) {
				return nil, types.ErrNotTokenDeposit.Wrapf("query doesn't exist plus not a bridge deposit")
			} else {
				return nil, err
			}
		}
		err = k.keeper.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		if err != nil {
			return nil, err
		}
	}
	// todo: should we remove this and allow reporters to overwrite their commits?
	has, err := k.keeper.Commits.Has(ctx, collections.Join(reporterAddr.Bytes(), query.Id))
	if err != nil {
		return nil, err
	}
	if has {
		return nil, fmt.Errorf("reporter already committed for the following query: %d", query.Id)
	}

	if query.QueryType == TRBBridgeQueryType {
		err = k.keeper.HandleBridgeDepositCommit(ctx, query, reporterAddr, msg.Hash)
		if err != nil {
			return nil, err
		}
		return &types.MsgCommitReportResponse{CommitId: query.Id}, nil
	}
	if query.Expiration.Before(blockTime) {
		return nil, types.ErrCommitWindowExpired
	}

	commit := types.Commit{
		Reporter: msg.Creator,
		QueryId:  queryId,
		Hash:     msg.Hash,
		Incycle:  query.CycleList,
	}
	err = k.keeper.Commits.Set(ctx, collections.Join(reporterAddr.Bytes(), query.Id), commit)
	if err != nil {
		return nil, err
	}
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_commit",
			sdk.NewAttribute("reporter", reporterAddr.String()),
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("commit_id", strconv.FormatUint(query.Id, 10)),
		),
	})
	return &types.MsgCommitReportResponse{CommitId: query.Id}, nil
}
