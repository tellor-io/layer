package keeper

import (
	"context"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/tellor-io/layer/x/bridge/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Request attestations for a snapshot of an aggregate report or a no stake report.
func (k msgServer) RequestAttestations(ctx context.Context, msg *types.MsgRequestAttestations) (*types.MsgRequestAttestationsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// ValidateBasic replacement
	err := validateRequestAttestations(msg)
	if err != nil {
		return nil, err
	}

	queryId, err := hex.DecodeString(registrytypes.Remove0xPrefix(msg.QueryId))
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to decode query id", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	timestampInt, err := strconv.ParseUint(msg.Timestamp, 10, 64)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to parse timestamp", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	timestamp := time.UnixMilli(int64(timestampInt))
	err = k.Keeper.CreateSnapshot(sdkCtx, queryId, timestamp, true)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to create snapshot", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"attestations_request",
			sdk.NewAttribute("query_id", msg.QueryId),
			sdk.NewAttribute("timestamp", msg.Timestamp),
		),
	})
	return &types.MsgRequestAttestationsResponse{}, nil
}

func validateRequestAttestations(msg *types.MsgRequestAttestations) error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
