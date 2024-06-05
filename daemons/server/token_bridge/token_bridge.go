package token_bridge

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/tellor-io/layer/daemons/server/types"
	tokenbridgeservertypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"

	"github.com/cosmos/cosmos-sdk/client"
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
	deposits, err := s.pendingDeposits.GetOldestReport()
	if err != nil {
		return nil, err
	}

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
	if err := types.RegisterTokenBridgeServiceHandlerClient(context.Background(), mux, types.NewTokenBridgeServiceClient(clientCtx)); err != nil {
		panic(err)
	}
}
