package median

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	gogogrpc "github.com/gogo/protobuf/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/daemons/server/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

var _ types.MedianValuesServiceServer = &medianServer{}

type medianServer struct {
	clientCtx        client.Context
	marketToExchange *pricefeedservertypes.MarketToExchangePrices
	marketParams     []pricefeedtypes.MarketParam
}

func NewMedianValuesServer(clientCtx client.Context, marketToExchange *pricefeedservertypes.MarketToExchangePrices, marketParams []pricefeedtypes.MarketParam) types.MedianValuesServiceServer {
	return &medianServer{
		clientCtx:        clientCtx,
		marketToExchange: marketToExchange,
		marketParams:     marketParams,
	}
}

func (s *medianServer) GetMedianValues(ctx context.Context, req *types.GetMedianValuesRequest) (*types.GetMedianValuesResponse, error) {
	values := s.marketToExchange.GetValidMedianPrices(s.marketParams, time.Now())
	medianValues := make([]*types.MedianValues, 0, len(values))
	for i, value := range values {
		medianValues = append(medianValues, &types.MedianValues{
			MarketId: i,
			Price:    value,
		})
	}
	return &types.GetMedianValuesResponse{MedianValues: medianValues}, nil

}

func StartMedianServer(
	clientCtx client.Context,
	server gogogrpc.Server,
	mux *runtime.ServeMux,
	marketParams []pricefeedtypes.MarketParam,
	marketToExchange *pricefeedservertypes.MarketToExchangePrices,
) {
	types.RegisterMedianValuesServiceServer(server, NewMedianValuesServer(clientCtx, marketToExchange, marketParams))
	types.RegisterMedianValuesServiceHandlerClient(context.Background(), mux, types.NewMedianValuesServiceClient(clientCtx))
}
