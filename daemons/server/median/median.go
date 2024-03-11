package median

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/daemons/server/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

var _ types.MedianValuesServiceServer = &medianServer{}

type medianServer struct {
	clientCtx               client.Context
	marketToExchange        *pricefeedservertypes.MarketToExchangePrices
	marketParams            []pricefeedtypes.MarketParam
	queryDataTomarketParams map[string]pricefeedtypes.MarketParam
}

func NewMedianValuesServer(clientCtx client.Context, marketToExchange *pricefeedservertypes.MarketToExchangePrices, marketParams []pricefeedtypes.MarketParam) types.MedianValuesServiceServer {
	mapQueryDataToMarketParams := make(map[string]pricefeedtypes.MarketParam)
	for _, marketParam := range marketParams {
		marketParam.QueryData = strings.ToLower(marketParam.QueryData)
		mapQueryDataToMarketParams[marketParam.QueryData] = marketParam
	}
	return &medianServer{
		clientCtx:               clientCtx,
		marketToExchange:        marketToExchange,
		marketParams:            marketParams,
		queryDataTomarketParams: mapQueryDataToMarketParams,
	}
}

func (s *medianServer) GetAllMedianValues(ctx context.Context, req *types.GetAllMedianValuesRequest) (*types.GetAllMedianValuesResponse, error) {
	values := s.marketToExchange.GetValidMedianPrices(s.marketParams, time.Now())
	medianValues := make([]*types.MedianValues, 0, len(values))
	for i, value := range values {
		medianValues = append(medianValues, &types.MedianValues{
			MarketId: i,
			Price:    value,
			// TODO: add exponent as well here
			// Exponent: ,
		})
	}
	return &types.GetAllMedianValuesResponse{MedianValues: medianValues}, nil

}

func (s *medianServer) GetMedianValue(ctx context.Context, req *types.GetMedianValueRequest) (*types.GetMedianValueResponse, error) {
	// check if query data exists in map
	mp, found := s.queryDataTomarketParams[req.QueryData]
	if !found {
		return nil, fmt.Errorf("no market param found for query data: %s", req.QueryData)
	}
	mv := s.marketToExchange.GetValidMedianPrices([]pricefeedtypes.MarketParam{mp}, time.Now())
	val, found := mv[mp.Id]
	if !found {
		return nil, fmt.Errorf("no median values found for query data: %s", req.QueryData)
	}
	res := types.MedianValues{
		// TODO: should market id be a string(querydata)?
		MarketId: mp.Id,
		Price:    val,
		Exponent: mp.Exponent,
	}
	return &types.GetMedianValueResponse{MedianValues: &res}, nil
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
