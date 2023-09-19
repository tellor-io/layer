package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"layer/x/querydatastorage/types"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetQueryData(goCtx context.Context, req *types.QueryGetQueryDataRequest) (*types.QueryGetQueryDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	qIdBytes, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query ID string: %v", err)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueryDataKey))
	queryData := store.Get(qIdBytes)
	return &types.QueryGetQueryDataResponse{QueryData: bytes.HexBytes(queryData).String()}, nil
}
