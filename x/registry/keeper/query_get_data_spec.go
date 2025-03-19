package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetDataSpec(goCtx context.Context, req *types.QueryGetDataSpecRequest) (*types.QueryGetDataSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.QueryType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request; query type cannot be empty")
	}

	exists, err := k.Keeper.HasSpec(goCtx, req.QueryType)
	if !exists {
		return nil, status.Error(codes.NotFound, "data spec not registered")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	dataSpec, err := k.Keeper.GetSpec(goCtx, req.QueryType)
	if err != nil {
		return nil, err
	}
	return &types.QueryGetDataSpecResponse{Spec: &dataSpec}, nil
}

func (k Querier) GetAllDataSpecs(goCtx context.Context, req *types.QueryGetAllDataSpecsRequest) (*types.QueryGetAllDataSpecsResponse, error) {
	specs, err := k.Keeper.GetAllDataSpecs(goCtx)
	if err != nil {
		return nil, err
	}

	// convert []types.DataSpec to []*types.DataSpec
	specPointers := make([]*types.DataSpec, len(specs))
	for i := range specs {
		specPointers[i] = &specs[i]
	}

	return &types.QueryGetAllDataSpecsResponse{Specs: specPointers}, nil
}
