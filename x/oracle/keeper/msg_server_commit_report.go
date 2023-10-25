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

func (k Keeper) GetSignature(ctx sdk.Context, reporter string, queryId []byte) (*types.CommitValue, error) {

	commitStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CommitReportKey))
	commit := commitStore.Get(append([]byte(reporter), queryId...))
	if commit == nil {
		return nil, status.Error(codes.NotFound, "no commits to reveal found")
	}
	var commitValue types.CommitValue
	k.cdc.Unmarshal(commit, &commitValue)
	return &commitValue, nil
}
