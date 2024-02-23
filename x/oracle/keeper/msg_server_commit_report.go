package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) CommitReport(goCtx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	queryId, err := utils.QueryIDFromDataString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query data: %s", err))
	}

	currentCycleQuery, err := k.GetCurrentQueryInCycleList(ctx)
	if err != nil {
		return nil, err
	}
	tip := k.GetQueryTip(ctx, queryId)

	if currentCycleQuery != msg.QueryData && tip.Amount.IsZero() {
		return nil, status.Error(codes.Unavailable, "query does not have tips and is not in cycle")
	}

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

	report := types.CommitReport{
		Report: &types.Commit{
			Creator: msg.Creator,
			QueryId: queryId,
			Hash:    msg.Hash,
		},
		Block: ctx.BlockHeight(),
	}
	err = k.Commits.Set(ctx, collections.Join(reporterAddr.Bytes(), queryId), report)
	if err != nil {
		return nil, err
	}

	return &types.MsgCommitReportResponse{}, nil
}
