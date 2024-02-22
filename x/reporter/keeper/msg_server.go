package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) WithdrawDelegatorReward(ctx context.Context, msg *types.MsgWithdrawDelegatorReward) (*types.MsgWithdrawDelegatorRewardResponse, error) {
	reporterVal := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)

	amount, err := k.WithdrawDelegationRewards(ctx, reporterVal.Bytes(), delAddr)
	if err != nil {
		return nil, err
	}

	return &types.MsgWithdrawDelegatorRewardResponse{Amount: amount}, nil
}

func (k msgServer) WithdrawReporterCommission(ctx context.Context, msg *types.MsgWithdrawReporterCommission) (*types.MsgWithdrawReporterCommissionResponse, error) {
	reporterVal := sdk.MustAccAddressFromBech32(msg.ReporterAddress)

	amount, err := k.Keeper.WithdrawReporterCommission(ctx, reporterVal.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.MsgWithdrawReporterCommissionResponse{Amount: amount}, nil
}
