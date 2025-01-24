package server

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/daemons/server/types"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge_tips"
)

// TokenBridgeTipServer defines the fields required for token bridge tip reports.
type TokenBridgeTipServer struct {
	tokenBridgeTipsCache *tokenbridgetypes.DepositTips
}

// WithTokenBridgeTipsCache sets the `tokenTipsCache` field.
// This is used by the token bridge tip service to communicate tip reports
// to the main application.
func (server *Server) WithTokenBridgeTipsCache(
	tokenTipsCache *tokenbridgetypes.DepositTips,
) *Server {
	server.tokenBridgeTipsCache = tokenTipsCache
	return server
}

func (s *TokenBridgeTipServer) GetPendingDepositTip(
	ctx context.Context,
	req *types.GetTokenBridgeTipRequest,
) (
	response *types.GetTokenBridgeTipResponse,
	err error,
) {
	// This panic is an unexpected condition because we initialize the market price cache in app initialization before
	// starting the server or daemons.
	if s.tokenBridgeTipsCache == nil {
		panic(fmt.Errorf("server not initialized correctly, tokenTipsCache not initialized"))
	}

	tip, err := s.tokenBridgeTipsCache.GetOldestTip()
	if err != nil {
		return nil, err
	}
	queryData := tip.QueryData

	return &types.GetTokenBridgeTipResponse{QueryData: queryData}, nil
}
