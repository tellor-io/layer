package token_bridge

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/tellor-io/layer/daemons/server/types"
	tokenbridgeservertypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
)

var _ types.TokenBridgeServiceServer = &tokenBridgeServer{}

type tokenBridgeServer struct {
	pendingDeposits *tokenbridgeservertypes.DepositReports
}

func NewTokenBridgeServer(pendingDeposits *tokenbridgeservertypes.DepositReports) types.TokenBridgeServiceServer {
	return &tokenBridgeServer{
		pendingDeposits: pendingDeposits,
	}
}

func (s *tokenBridgeServer) GetPendingDepositReport(ctx context.Context, req *types.GetPendingDepositReportRequest) (*types.GetPendingDepositReportResponse, error) {
	// Retrieve pending deposits from s.pendingDeposits
	deposits := s.pendingDeposits.GetOldestReport()

	// Create response with pending deposits
	response := &types.GetPendingDepositReportResponse{
		QueryData: deposits.QueryData,
		Value:     deposits.Value,
	}

	return response, nil
}

func StartTokenBridgeServer(
	clientCtx client.Context,
	server gogogrpc.Server,
	mux *runtime.ServeMux,
	pendingDeposits *tokenbridgeservertypes.DepositReports,
) {
	types.RegisterTokenBridgeServiceServer(server, NewTokenBridgeServer(pendingDeposits))
	types.RegisterTokenBridgeServiceHandlerClient(context.Background(), mux, types.NewTokenBridgeServiceClient(clientCtx))
}
