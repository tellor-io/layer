package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	rk "github.com/tellor-io/layer/x/registry/keeper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) CommitReport(goCtx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	tip, _ := k.GetQueryTips(ctx, k.TipStore(ctx), msg.QueryData)
	if !k.Keeper.GetBlockTips(ctx, ctx.BlockHeight()-1).Tips[msg.QueryData] && tip.Amount.IsZero() {
		return nil, status.Error(codes.Unavailable, "query data does not have tips/in cycle")
	}
	reporter := sdk.MustAccAddressFromBech32(msg.Creator)

	// get delegation info
	validator := k.stakingKeeper.Validator(ctx, sdk.ValAddress(reporter))
	// check if msg sender is validator
	if !reporter.Equals(validator.GetOperator()) {
		return nil, status.Error(codes.Unauthenticated, "sender is not validator")
	}

	if rk.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	queryData, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query data: %s", err))
	}
	queryId := HashQueryData(queryData)
	report := types.CommitReport{
		Report: &types.Commit{
			Creator:   msg.Creator,
			QueryId:   queryId,
			Signature: msg.Signature,
		},
		Block: ctx.BlockHeight(),
	}
	k.SetCommitReport(ctx, reporter, &report)

	return &types.MsgCommitReportResponse{}, nil
}
