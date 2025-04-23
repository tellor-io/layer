package token_bridge

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	types "github.com/tellor-io/layer/daemons/server/types/daemons"
	tokenbridgeservertypes "github.com/tellor-io/layer/daemons/server/types/token_bridge_tips"

	"github.com/cosmos/cosmos-sdk/client"
)

var _ types.TokenBridgeTipServiceServer = &tokenBridgeTipServer{}

type tokenBridgeTipServer struct {
	tips *tokenbridgeservertypes.DepositTips
}

func NewTokenBridgeTipServer(tips *tokenbridgeservertypes.DepositTips) types.TokenBridgeTipServiceServer {
	return &tokenBridgeTipServer{
		tips: tips,
	}
}

func (s *tokenBridgeTipServer) GetTokenBridgeTip(ctx context.Context, req *types.GetTokenBridgeTipRequest) (*types.GetTokenBridgeTipResponse, error) {
	// Retrieve pending deposits from s.pendingDeposits
	tip, err := s.tips.GetOldestTip()
	if err != nil {
		return nil, err
	}

	// Create response with pending deposits
	response := &types.GetTokenBridgeTipResponse{
		QueryData: tip.QueryData,
	}

	return response, nil
}

func StartTokenBridgeTipsServer(
	clientCtx client.Context,
	server gogogrpc.Server,
	mux *runtime.ServeMux,
	tips *tokenbridgeservertypes.DepositTips,
) {
	types.RegisterTokenBridgeTipServiceServer(server, NewTokenBridgeTipServer(tips))
	if err := types.RegisterTokenBridgeTipServiceHandlerClient(context.Background(), mux, types.NewTokenBridgeTipServiceClient(clientCtx)); err != nil {
		panic(err)
	}
}
