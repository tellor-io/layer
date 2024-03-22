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

func (k msgServer) RegisterOperatorPubkey(ctx context.Context, msg *types.MsgRegisterOperatorPubkey) (*types.MsgRegisterOperatorPubkeyResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	operatorAddr2, err := convertPrefix(msg.Creator, "tellorvaloper")
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to convert operator address prefix", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

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

	error := k.Keeper.SetEVMAddressByOperator(sdkCtx, operatorAddr2, ethAddress)
	if error != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to set eth address by operator", "error", error)
		return nil, status.Error(codes.Internal, error.Error())
	}

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
