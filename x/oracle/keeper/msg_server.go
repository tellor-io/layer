package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type msgServer struct {
	keeper Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) BatchSubmitValue(ctx context.Context, msg *types.MsgBatchSubmitValue) (res *types.MsgBatchSubmitValueResponse, err error) {
	// validate reporter address and convert from bech32 to AccAddress
	reporterAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// check that the length of operations is less than or equal to the max batch size
	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	reporterStake, delegationsUsed, err := k.keeper.reporterKeeper.GetReporterStake(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}
	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporterStake, params.MinStakeAmount)
	}
	reportingPower := reporterStake.Quo(layertypes.PowerReduction).Uint64()
	// maxBatchSize := params.MaxBatchSize
	// TODO: for now, hardcode max batch size to 20
	maxBatchSize := 20
	if len(msg.Values) > maxBatchSize {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "too many reports in batch, max is %d", maxBatchSize)
	}
	failedIndices := []uint32{}
	for i, singleValue := range msg.Values {
		// validate each individual report
		queryDataBz := singleValue.QueryData
		value := singleValue.Value
		if len(queryDataBz) == 0 || singleValue.Value == "" {
			failedIndices = append(failedIndices, uint32(i))
			continue
		}
		isTokenBridgeDeposit, err := k.keeper.PreventBridgeWithdrawalReport(ctx, queryDataBz)
		if err != nil {
			failedIndices = append(failedIndices, uint32(i))
			continue
		}
		queryId := utils.QueryIDFromData(queryDataBz)
		query, err := k.keeper.CurrentQuery(ctx, queryId)

		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				failedIndices = append(failedIndices, uint32(i))
				continue
			}
			if !isTokenBridgeDeposit {
				failedIndices = append(failedIndices, uint32(i))
				continue
			}
			query, err = k.keeper.TokenBridgeDepositQuery(ctx, queryDataBz)
			if err != nil {
				failedIndices = append(failedIndices, uint32(i))
				continue
			}
			// sets to storage
			err = k.keeper.HandleBridgeDepositDirectReveal(ctx, query, queryDataBz, reporterAddr, value, reportingPower)
			if err != nil {
				return nil, err
			}
		} else {
			// sets to storage
			err = k.keeper.DirectReveal(ctx, query, queryDataBz, value, reporterAddr, reportingPower, isTokenBridgeDeposit)
			if err != nil {
				failedIndices = append(failedIndices, uint32(i))
				continue
			}
		}
		// sets to storage
		// TODO: if this errors, we should probably fail the entire batch
		err = k.keeper.reporterKeeper.SetReporterStakeByQueryId(ctx, reporterAddr, delegationsUsed, reporterStake, queryId)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgBatchSubmitValueResponse{
		FailedIndices: failedIndices,
	}, nil
}
