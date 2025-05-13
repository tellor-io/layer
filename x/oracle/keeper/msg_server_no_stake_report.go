package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) NoStakeReport(ctx context.Context, msg *types.MsgNoStakeReport) (res *types.MsgNoStakeReportResponse, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := k.validateNoStakeReport(msg); err != nil {
		return nil, err
	}

	reporterAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	queryData := msg.QueryData
	value := msg.Value
	// Size limit check (0.5MB)
	limit, err := k.keeper.QueryDataLimit.Get(ctx)
	if err != nil {
		return nil, err
	}
	if len(queryData) > int(limit.Limit) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query data too large")
	}

	timestamp := sdkCtx.BlockTime()
	queryId := utils.QueryIDFromData(queryData)

	err = k.keeper.NoStakeReports.Set(sdkCtx, collections.Join3(queryId, reporterAddr.Bytes(), uint64(timestamp.UnixMilli())), types.NoStakeMicroReport{
		Reporter:    reporterAddr.String(),
		QueryData:   queryData,
		Value:       value,
		Timestamp:   timestamp,
		BlockNumber: uint64(sdkCtx.BlockHeight()),
	})
	if err != nil {
		return nil, err
	}

	// add queryId to tracker
	err = k.keeper.NoStakeTracker.Set(sdkCtx, collections.Join(uint64(sdkCtx.BlockHeight()), queryId), reporterAddr.Bytes())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (k msgServer) validateNoStakeReport(msg *types.MsgNoStakeReport) error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// make sure query data is not empty
	if len(msg.QueryData) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "query data cannot be empty")
	}
	// make sure value is not empty
	if msg.Value == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be empty")
	}
	return nil
}
