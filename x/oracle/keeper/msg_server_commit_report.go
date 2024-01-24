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

	reporter := sdk.MustAccAddressFromBech32(msg.Creator)

	// get delegation info
	validator, err := k.stakingKeeper.Validator(ctx, sdk.ValAddress(reporter))
	if err != nil {
		return nil, err
	}
	// check if msg sender is validator
	if msg.Creator != validator.GetOperator() {
		return nil, status.Error(codes.Unauthenticated, "sender is not validator")
	}
	// check if validator is bonded
	_, isBonded := k.IsReporterStaked(ctx, sdk.ValAddress(reporter))
	if !isBonded {
		return nil, types.ErrValidatorNotBonded
	}

	// check if querydata has prefix 0x
	if rk.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	// decode query data hex string to bytes
	queryData, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query data: %s", err))
	}
	queryId := HashQueryData(queryData)
	report := types.CommitReport{
		Report: &types.Commit{
			Creator:     msg.Creator,
			QueryId:     queryId,
			SaltedValue: msg.SaltedValue,
		},
		Block: ctx.BlockHeight(),
	}
	k.SetCommitReport(ctx, reporter, &report)

	return &types.MsgCommitReportResponse{}, nil
}
