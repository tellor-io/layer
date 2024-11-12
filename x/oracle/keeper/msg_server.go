package keeper

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/tellor-io/layer/x/oracle/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k msgServer) SendQueryGetCurrentAggregatedReport(goCtx context.Context, msg *types.MsgSendQueryGetCurrentAggregatedReport) (*types.MsgSendQueryetCurrentAggregatedReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	chanCap, found := k.keeper.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(k.keeper.GetPort(ctx), msg.ChannelId))
	if !found {
		return nil, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}
	q := types.QueryGetCurrentAggregateReportRequest{
		QueryId: msg.QueryId,
	}
	reqs := []abci.RequestQuery{
		{
			Path: "/layer.oracle.Query/GetCurrentAggregateReport",
			Data: k.keeper.cdc.MustMarshal(&q),
		},
	}
	timeoutTimestamp := ctx.BlockTime().Add(time.Minute).UnixNano()
	seq, err := k.keeper.SendQuery(ctx, types.PortID, msg.ChannelId, chanCap, reqs, clienttypes.ZeroHeight(), uint64(timeoutTimestamp))
	if err != nil {
		return nil, err
	}

	return &types.MsgSendQueryetCurrentAggregatedReportResponse{
		Sequence: seq,
	}, nil
}
