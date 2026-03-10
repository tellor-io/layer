package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) NoStakeReport(ctx context.Context, msg *types.MsgNoStakeReport) (res *types.MsgNoStakeReportResponse, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	reporterAddr, err := k.validateNoStakeReport(msg)
	if err != nil {
		return nil, err
	}

	queryData := msg.QueryData
	value := registrytypes.Remove0xPrefix(msg.Value)
	timestamp := sdkCtx.BlockTime().UnixMilli()
	queryId := utils.QueryIDFromData(queryData)

	// check if report for this queryId already exists at this height
	exists, err := k.keeper.NoStakeReports.Has(sdkCtx, collections.Join(queryId, uint64(timestamp)))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}
	if exists {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "report for this queryId already exists at this height, please resubmit next block")
	}

	// check if queryId:queryData map is already set
	exists, err = k.keeper.NoStakeReportedQueries.Has(sdkCtx, queryId)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = k.keeper.NoStakeReportedQueries.Set(sdkCtx, queryId, queryData)
		if err != nil {
			return nil, err
		}
	}

	// Size limit check (0.5MB)
	limit, err := k.keeper.QueryDataLimit.Get(ctx)
	if err != nil {
		return nil, err
	}
	if len(queryData) > int(limit.Limit) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query data too large")
	}

	err = k.keeper.NoStakeReports.Set(sdkCtx, collections.Join(queryId, uint64(timestamp)), types.NoStakeMicroReport{
		Reporter:    reporterAddr,
		Value:       value,
		Timestamp:   sdkCtx.BlockTime(),
		BlockNumber: uint64(sdkCtx.BlockHeight()),
	})
	if err != nil {
		return nil, err
	}

	return &types.MsgNoStakeReportResponse{}, nil
}

func (k msgServer) validateNoStakeReport(msg *types.MsgNoStakeReport) (reporter sdk.AccAddress, err error) {
	reporter, err = sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	// make sure query data is not empty
	if len(msg.QueryData) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "query data cannot be empty")
	}
	// make sure value is not empty
	if msg.Value == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be empty")
	}
	return reporter, nil
}
