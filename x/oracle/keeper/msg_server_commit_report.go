package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	registryKeeper "github.com/tellor-io/layer/x/registry/keeper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) CommitReport(goCtx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid sender address: %v", err))
	}
	valAddr := sdk.ValAddress(delAddr)

	// get delegation info
	validator := k.stakingKeeper.Validator(ctx, valAddr)
	// check if msg sender is validator
	if !delAddr.Equals(validator.GetOperator()) {
		return nil, status.Error(codes.Unauthenticated, "sender is not validator")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CommitReportKey))
	if registryKeeper.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	queryData, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query data: %s", err))
	}
	queryId := HashQueryData(queryData)
	report := types.CommitValue{
		Report: &types.Commit{
			Creator:   msg.Creator,
			QueryId:   queryId,
			Signature: msg.Signature,
		},
		Block: ctx.BlockHeight(),
	}
	store.Set(append([]byte(msg.Creator), queryId...), k.cdc.MustMarshal(&report))

	return &types.MsgCommitReportResponse{}, nil
}
