package keeper

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) RegisterSpec(goCtx context.Context, msg *types.MsgRegisterSpec) (*types.MsgRegisterSpecResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	err := validateRegisterSpec(msg)
	if err != nil {
		return nil, err
	}
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
			sdk.NewAttribute("report_buffer_window", fmt.Sprintf("%d", msg.Spec.ReportBlockWindow)),
		),
	})
	return &types.MsgRegisterSpecResponse{}, nil
}

func validateRegisterSpec(msg *types.MsgRegisterSpec) error {
	_, err := sdk.AccAddressFromBech32(msg.Registrar)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// querytype should be non-empty string
	if reflect.TypeOf(msg.QueryType).Kind() != reflect.String {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query type must be a string")
	}
	if msg.QueryType == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query type must be a non-empty string")
	}
	//  ensure correctness of data within the Spec
	if msg.Spec.AbiComponents == nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec abi components should not be empty")
	}
	if msg.Spec.AggregationMethod == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec aggregation method should not be empty")
	}
	if msg.Spec.Registrar == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec registrar should not be empty")
	}
	if msg.Spec.ResponseValueType == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "spec response value type should not be empty")
	}
	return nil
}
