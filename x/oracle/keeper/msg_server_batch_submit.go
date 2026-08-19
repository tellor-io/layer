package keeper

import (
	"context"
	"encoding/hex"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

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

	if len(msg.Values) == 0 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no values in batch")
	}

	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	// ReporterStake runs once on the first report that passes pre-reveal validation,
	// using that report's query id. Same stake/period lifecycle as MsgSubmitValue.
	var (
		reportingPower   uint64
		stakeInitialized bool
	)
	failedIndices := []uint32{}
	for i, singleValue := range msg.Values {
		queryDataBz := singleValue.QueryData
		valueBytes, err := hex.DecodeString(registrytypes.Remove0xPrefix(singleValue.Value))
		if err != nil {
			failedIndices = append(failedIndices, uint32(i))
			continue
		}
		value := hex.EncodeToString(valueBytes)
		if len(queryDataBz) == 0 || value == "" {
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

		bridgeDepositPath := false
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
			bridgeDepositPath = true
		}

		if !stakeInitialized {
			reporterStake, err := k.keeper.reporterKeeper.ReporterStake(ctx, reporterAddr, queryId)
			if err != nil {
				return nil, err
			}
			if reporterStake.LT(params.MinStakeAmount) {
				return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporterStake, params.MinStakeAmount)
			}
			reportingPower = reporterStake.Quo(layertypes.PowerReduction).Uint64()
			stakeInitialized = true
		}

		if bridgeDepositPath {
			err = k.keeper.HandleBridgeDepositDirectReveal(ctx, query, queryDataBz, reporterAddr, value, reportingPower)
			if err != nil {
				return nil, err
			}
		} else {
			err = k.keeper.DirectReveal(ctx, query, queryDataBz, value, reporterAddr, reportingPower, isTokenBridgeDeposit)
			if err != nil {
				failedIndices = append(failedIndices, uint32(i))
				continue
			}
		}
	}

	if len(failedIndices) == len(msg.Values) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "all reports in batch failed")
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
