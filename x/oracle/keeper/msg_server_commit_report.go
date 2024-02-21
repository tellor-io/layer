package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) CommitReport(goCtx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	queryId, err := utils.QueryIDFromDataString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query data: %s", err))
	}

	currentCycleQuery := k.GetCurrentQueryInCycleList(ctx)
	tip := k.GetQueryTip(ctx, queryId)
	if currentCycleQuery != msg.QueryData && tip.Amount.IsZero() {
		return nil, status.Error(codes.Unavailable, "query data does not have tips/not in cycle")
	}

	reporter := sdk.MustAccAddressFromBech32(msg.Creator)

	// get delegation info
	validator, err := k.stakingKeeper.Validator(ctx, sdk.ValAddress(reporter))
	if err != nil {
		return nil, err
	}
	if validator.IsJailed() {
		return nil, status.Error(codes.Unavailable, "validator is jailed")
	}
	if !validator.IsBonded() {
		return nil, status.Error(codes.Unavailable, "validator is not bonded")
	}
	report := types.CommitReport{
		Report: &types.Commit{
			Creator: msg.Creator,
			QueryId: queryId,
			Hash:    msg.Hash,
		},
		Block: ctx.BlockHeight(),
	}
	err = k.Commits.Set(ctx, collections.Join(reporter.Bytes(), queryId), report)
	if err != nil {
		return nil, err
	}

	return &types.MsgCommitReportResponse{}, nil
}
