package server

import (
	"context"
	"fmt"

	// pricefeedmetrics "github.com/tellor-io/layer/daemons/pricefeed/metrics"

	// pricefeedmetrics "github.com/tellor-io/layer/daemons/pricefeed/metrics"

	"github.com/tellor-io/layer/daemons/server/types"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
)

// TokenBridgeFeedServer defines the fields required for price updates.
type TokenBridgeFeedServer struct {
	tokenDepositsCache *tokenbridgetypes.DepositReports
}

// WithPriceFeedMarketToExchangePrices sets the `MarketToExchangePrices` field.
// This is used by the price feed service to communicate price updates
// to the main application.
func (server *Server) WithTokenDepositsCache(
	tokenDepositsCache *tokenbridgetypes.DepositReports,
) *Server {
	server.tokenDepositsCache = tokenDepositsCache
	return server
}

func (s *TokenBridgeFeedServer) GetPendingDepositReport(
	ctx context.Context,
	req *types.GetPendingDepositReportRequest,
) (
	response *types.GetPendingDepositReportResponse,
	err error,
) {

	// This panic is an unexpected condition because we initialize the market price cache in app initialization before
	// starting the server or daemons.
	if s.tokenDepositsCache == nil {
		panic(fmt.Errorf("server not initialized correclty, tokenDepositsCache not initialized"))
	}

	report, err := s.tokenDepositsCache.GetOldestReport()
	if err != nil {
		return nil, err
	}
	queryData := report.QueryData
	value := report.Value

	return &types.GetPendingDepositReportResponse{QueryData: queryData, Value: value}, nil
}
