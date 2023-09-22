package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetReportsbyQid(goCtx context.Context, req *types.QueryGetReportsbyQidRequest) (*types.QueryGetReportsbyQidResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	qIdBytes, err := hex.DecodeString(req.QId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query ID string: %v", err)
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
	reportsBytes := store.Get(qIdBytes)
	var reports types.Reports
	if err := k.cdc.Unmarshal(reportsBytes, &reports); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}
	return &types.QueryGetReportsbyQidResponse{Reports: reports}, nil
}
