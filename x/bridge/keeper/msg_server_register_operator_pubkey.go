package keeper

import (
	"context"

	math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// from staking module:
// // ValAddress defines a wrapper around bytes meant to present a validator's
// // operator. When marshaled to a string or JSON, it uses Bech32.
// type ValAddress []byte

func (k msgServer) RegisterOperatorPubkey(ctx context.Context, msg *types.MsgRegisterOperatorPubkey) (*types.MsgRegisterOperatorPubkeyResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.Keeper.Logger(sdkCtx).Info("@FuncRegisterOperatorPubkey", "msg", msg)

	// operatorAddr1, err := sdk.ValAddressFromBech32(msg.Creator)
	// if err != nil {
	// 	k.Keeper.Logger(sdkCtx).Error("failed to get operator address", "error", err)
	// 	return nil, status.Error(codes.InvalidArgument, err.Error())
	// }
	// k.Keeper.Logger(sdkCtx).Info("operator address 1", "address", operatorAddr1)

	operatorAddr2, err := convertPrefix(msg.Creator, "tellorvaloper")
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to convert operator address prefix", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	k.Keeper.Logger(sdkCtx).Info("operator address 2", "address", operatorAddr2)

	operatorValAddr, err := sdk.ValAddressFromBech32(operatorAddr2)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to get operator address", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the creator is a staked validator
	validator, err := k.Keeper.stakingKeeper.GetValidator(sdkCtx, operatorValAddr)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to get validator", "error", err)
		return nil, status.Error(codes.PermissionDenied, "this validator not found")
	}
	consensusPower := uint64(validator.GetConsensusPower(math.NewInt(10)))
	if consensusPower == 0 {
		k.Keeper.Logger(sdkCtx).Error("validator is not staked", "error", err)
		return nil, status.Error(codes.PermissionDenied, "this validator is not staked")
	}

	// get eth address
	ethAddress, err := k.Keeper.EVMAddressFromSignature(sdkCtx, msg.OperatorPubkey)
	if err != nil {
		k.Logger(sdkCtx).Error("failed to get eth address from operator pubkey", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	k.Keeper.Logger(sdkCtx).Info("eth address", "address", ethAddress)

	// // Ensure the pubkey is not already registered
	// if k.GetOperatorPubKey(sdkCtx, msg.PubKey) {
	// 	return nil, status.Error(codes.AlreadyExists, "pubkey is already registered")
	// }

	// // Add the pubkey to the store
	// k.SetOperatorPubKey(sdkCtx, msg.PubKey, msg.Creator)

	return &types.MsgRegisterOperatorPubkeyResponse{}, nil
}

// convertPrefix converts the prefix of a bech32 encoded address
func convertPrefix(address, newPrefix string) (string, error) {
	_, bz, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return "", err
	}
	newAddr, err := bech32.ConvertAndEncode(newPrefix, bz)
	if err != nil {
		return "", err
	}
	return newAddr, nil
}
