package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) RegisterSpec(goCtx context.Context, msg *types.MsgRegisterSpec) (*types.MsgRegisterSpecResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	specExists, _ := k.SpecRegistry.Has(ctx, msg.QueryType)
	if specExists {
		return nil, status.Error(codes.AlreadyExists, "spec already exists")
	}
	if !SupportedType(msg.Spec.ValueType) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("value type not supported: %s", msg.Spec.ValueType))
	}
	if err := k.SpecRegistry.Set(ctx, msg.QueryType, msg.Spec); err != nil {
		return nil, err
	}

	return &types.MsgRegisterSpecResponse{}, nil
}

func SupportedType(dataType string) bool {
	switch dataType {
	case "string", "bool", "address", "bytes":
		return true
	case "int8", "int16", "int32", "int64", "int128", "int256":
		return true
	case "uint8", "uint16", "uint32", "uint64", "uint128", "uint256":
		return true
	default:
		return false
	}
}
