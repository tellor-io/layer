package keeper

import (
	"context"
	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) CommitReport(goCtx context.Context, msg *types.MsgCommitReport) (*types.MsgCommitReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CommitReportKey))
	report := types.CommitValue{
		Report: msg,
		Block:  ctx.BlockHeight(),
	}
	qIdBytes, err := hex.DecodeString(msg.QueryId)
	if err != nil {
		return nil, err
	}
	store.Set(append([]byte(msg.Creator), qIdBytes...), k.cdc.MustMarshal(&report))

	return &types.MsgCommitReportResponse{}, nil
}

func (k Keeper) getCommit(ctx sdk.Context, reporter, queryId string) (*types.CommitValue, error) {

	commitStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CommitReportKey))
	qIdBytes, err := hex.DecodeString(queryId)
	if err != nil {
		return nil, err
	}
	commit := commitStore.Get(append([]byte(reporter), qIdBytes...))
	if commit == nil {
		return nil, status.Error(codes.NotFound, "no commits to reveal found")
	}
	var commitValue types.CommitValue
	k.cdc.Unmarshal(commit, &commitValue)
	return &commitValue, nil
}
