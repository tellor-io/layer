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

func (k msgServer) BatchSubmitValue(ctx context.Context, msg *types.MsgBatchSubmitValue) (res *types.MsgBatchSubmitValueResponse, err error) {
	// also validates reporter address and convert from bech32 to AccAddress
	reporterAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	reporterStake, delegationsUsed, _, _, err := k.keeper.reporterKeeper.GetReporterStake(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}
	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporterStake, params.MinStakeAmount)
	}
	reportingPower := reporterStake.Quo(layertypes.PowerReduction).Uint64()

	maxBatchSize, err := k.keeper.GetMaxBatchSize(ctx)
	if err != nil {
		return nil, err
	}

	if maxBatchSize == 0 {
		maxBatchSize = 20
	}

	if len(msg.Values) > int(maxBatchSize) {
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
			// sets to storage if no error
			err = k.keeper.HandleBridgeDepositDirectReveal(ctx, query, queryDataBz, reporterAddr, value, reportingPower)
			if err != nil {
				return nil, err
			}
		} else {
			// sets to storage if no error
			err = k.keeper.DirectReveal(ctx, query, queryDataBz, value, reporterAddr, reportingPower, isTokenBridgeDeposit)
			if err != nil {
				failedIndices = append(failedIndices, uint32(i))
				continue
			}
		}
		// sets to storage if no error
		// fails transaction if error happens at this point since the above operations are setting to storage
		// and we should revert if this fails.
		// plus this should never fail since we all it does is set the reporter stake for the given queryId
		err = k.keeper.reporterKeeper.SetReporterStakeByQueryId(ctx, reporterAddr, delegationsUsed, reporterStake, queryId)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgBatchSubmitValueResponse{
		FailedIndices: failedIndices,
	}, nil
}

func (k msgServer) UpdateMaxBatchSize(ctx context.Context, req *types.MsgUpdateMaxBatchSize) (*types.MsgUpdateMaxBatchSizeResponse, error) {
	if k.keeper.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.keeper.GetAuthority(), req.Authority)
	}
	size := req.MaxBatchSize
	if size == 0 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "max batch size must be greater than 0")
	}
	if err := k.keeper.MaxBatchSize.Set(ctx, size); err != nil {
		return nil, err
	}

	return &types.MsgUpdateMaxBatchSizeResponse{}, nil
}
