package keeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"register_data_spec",
			sdk.NewAttribute("registrar", msg.Registrar),
			sdk.NewAttribute("document_hash_id", msg.Spec.DocumentHash),
			sdk.NewAttribute("query_type", msg.QueryType),
			sdk.NewAttribute("aggregate_method", msg.Spec.AggregationMethod),
			sdk.NewAttribute("response_value_type", msg.Spec.ResponseValueType),
			sdk.NewAttribute("report_buffer_window", msg.Spec.ReportBufferWindow.String()),
		),
	})
	return &types.MsgRegisterSpecResponse{}, nil
}
