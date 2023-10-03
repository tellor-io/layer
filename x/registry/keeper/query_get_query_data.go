package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/registry/types"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetQueryData(goCtx context.Context, req *types.QueryGetQueryDataRequest) (*types.QueryGetQueryDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	queryData, err := k.QueryData(ctx, req.QueryId)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetQueryDataResponse{QueryData: bytes.HexBytes(queryData).String()}, nil
}

func (k Keeper) QueryData(ctx sdk.Context, queryId string) ([]byte, error) {
	if !IsQueryIdValid(queryId) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query id: %s", queryId))
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueryRegistryKey))
	qIdBytes := common.HexToHash(queryId).Bytes()

	queryData := store.Get(qIdBytes)
	if queryData == nil {
		return nil, status.Error(codes.NotFound, "query data not found")
	}

	return queryData, nil
}

// has0xPrefix validates str begins with '0x' or '0X'.
// From: https://github.com/ethereum/go-ethereum/blob/5c6f4b9f0d4270fcc56df681bf003e6a74f11a6b/common/bytes.go#L51
func Has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// check queryId is valid ie 32 bytes
func IsQueryIdValid(queryId string) bool {
	hasPrefix := Has0xPrefix(queryId)
	if hasPrefix {
		queryId = queryId[2:]
	}
	if len(queryId) != 64 {
		return false
	}
	return true
}
