package keeper

import (
	"context"
	"errors"
	"strings"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) SubmitValue(ctx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}

	err = k.keeper.PreventBridgeWithdrawalReport(msg.QueryData)
	if err != nil {
		return nil, err
	}

	// get reporter
	reporterStake, err := k.keeper.reporterKeeper.ReporterStake(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}
	params, err := k.keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if reporterStake.LT(params.MinStakeAmount) {
		return nil, errorsmod.Wrapf(types.ErrNotEnoughStake, "reporter has %s, required %s", reporterStake, params.MinStakeAmount)
	}

	votingPower := reporterStake.Quo(layertypes.PowerReduction).Uint64()
	// decode query data hex string to bytes

	queryId := utils.QueryIDFromData(msg.QueryData)

	query, err := k.keeper.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		query, err = k.keeper.TokenBridgeDepositCheck(ctx, msg.QueryData)
		if err != nil {
			return nil, err
		}

		err = k.keeper.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		if err != nil {
			return nil, err
		}
		err = k.keeper.HandleBridgeDepositDirectReveal(ctx, query, msg.QueryData, reporterAddr, msg.Value, votingPower)
		if err != nil {
			return nil, err
		}
		return &types.MsgSubmitValueResponse{}, nil
	}
	isBridgeDeposit := strings.EqualFold(query.QueryType, TRBBridgeQueryType)
	err = k.keeper.DirectReveal(ctx, query, msg.QueryData, msg.Value, reporterAddr, votingPower, isBridgeDeposit)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitValueResponse{}, nil
}

func (k Keeper) DirectReveal(ctx context.Context,
	query types.QueryMeta,
	qDataBytes []byte,
	value string,
	reporterAddr sdk.AccAddress,
	votingPower uint64,
	bridgeDeposit bool,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	if bridgeDeposit {
		return k.HandleBridgeDepositDirectReveal(ctx, query, qDataBytes, reporterAddr, value, votingPower)
	}

	if query.Amount.IsZero() && !query.CycleList {
		return types.ErrNoTipsNotInCycle
	}

	offset, err := k.GetReportOffsetParam(ctx)
	if err != nil {
		return err
	}

	if query.Expiration+offset <= uint64(blockHeight) {
		return types.ErrSubmissionWindowExpired
	}

	return k.SetValue(ctx, reporterAddr, query, value, qDataBytes, votingPower, query.CycleList)
}
