package keeper

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) RegisterSpec(goCtx context.Context, msg *types.MsgRegisterSpec) (*types.MsgRegisterSpecResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	specExists, _ := k.Keeper.HasSpec(ctx, msg.QueryType)
	if specExists {
		return nil, status.Error(codes.AlreadyExists, "data spec previously registered")
	}
	msg.Spec.ResponseValueType = strings.ToLower(msg.Spec.ResponseValueType)
	msg.Spec.AggregationMethod = strings.ToLower(msg.Spec.AggregationMethod)
	msg.Spec.Registrar = msg.Registrar

	if !types.SupportedType(msg.Spec.ResponseValueType) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("value type not supported: %s", msg.Spec.ResponseValueType))
	}
	// TODO: assert the aggregation can be handled
	if !types.SupportedAggregationMethod[msg.Spec.AggregationMethod] {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("aggregation method not supported: %s", msg.Spec.AggregationMethod))
	}

	if err := k.Keeper.SetDataSpec(ctx, msg.QueryType, msg.Spec); err != nil {
		return nil, err
	}

	return &types.MsgRegisterSpecResponse{}, nil
}
