package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) CommitReport(goCtx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Check if query data begins with 0x and remove it
	msg.QueryData = regtypes.Remove0xPrefix(msg.QueryData)
	// Try to decode query data from hex string
	queryData, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query data: %s", err))
	}
	tip, _ := k.GetQueryTips(ctx, k.TipStore(ctx), msg.QueryData)
	currentCycleQuery := k.GetCurrentQueryInCycleList(ctx)
	if currentCycleQuery != msg.QueryData && tip.Amount.IsZero() {
		return nil, status.Error(codes.Unavailable, "query data does not have tips/not in cycle")
	}
	reporter := sdk.MustAccAddressFromBech32(msg.Creator)

	// get delegation info
	validator, err := k.stakingKeeper.Validator(ctx, sdk.ValAddress(reporter))
	if err != nil {
		return nil, err
	}
	if !validator.IsBonded() {
		return nil, status.Error(codes.Unavailable, "validator is not bonded")
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
