package customquery

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tellor-io/layer/daemons/custom_query/contracts/contract_handlers"
	rpc_handler "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_handler"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

// Result holds the value returned from an endpoint
type Result struct {
	Value      float64
	Err        error
	EndpointID string
}

// FetchPriceResult holds the result of a price fetch operation
type FetchPriceResult struct {
	EncodedValue string
	RawResults   []Result
	QueryID      string
	ResponseType string
	SuccessRate  float64
}

// FetchPrice fetches price data for the given query ID
func FetchPrice(
	ctx context.Context,
	query QueryConfig,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (*FetchPriceResult, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	totalEndpoints := len(query.RpcReaders) + len(query.ContractReaders)
	results := make(chan Result, totalEndpoints)
	var wg sync.WaitGroup

	// Launch goroutines for contract endpoints
	for _, contractEndpoint := range query.ContractReaders {
		wg.Add(1)
		go func(ep ContractHandler) {
			defer wg.Done()
			result := fetchFromContractEndpoint(ctx, ep, priceCache)
			results <- result
		}(contractEndpoint)
	}
	// Launch goroutines for REST API endpoints
	for _, rpchandler := range query.RpcReaders {
		wg.Add(1)
		go func(ep RpcHandler) {
			defer wg.Done()
			result := fetchFromRpcEndpoint(ctx, ep, priceCache)
			results <- result
		}(rpchandler)

	}
	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []Result
	// Count successful results
	var successfulResults []Result
	for result := range results {
		allResults = append(allResults, result)
		if result.Err == nil {
			successfulResults = append(successfulResults, result)
		}
	}
	// Check if we have enough successful responses
	if len(successfulResults) < query.MinResponses {
		return nil, fmt.Errorf("insufficient successful responses: got %d, need %d",
			len(successfulResults), query.MinResponses)
	}
	fmt.Println("Successful results:", successfulResults)
	// Aggregate results
	aggregatedValue, err := aggregateResults(successfulResults, query.AggregationMethod, query.ResponseType)
	if err != nil {
		return nil, err
	}

	return &FetchPriceResult{
		EncodedValue: aggregatedValue,
		RawResults:   allResults,
		QueryID:      query.ID,
		ResponseType: query.ResponseType,
		SuccessRate:  float64(len(successfulResults)) / float64(totalEndpoints),
	}, nil
}

// fetchFromContractEndpoint fetches data from a smart contract
func fetchFromContractEndpoint(
	ctx context.Context,
	contractReader ContractHandler,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) Result {
	handler, err := contract_handlers.GetHandler(contractReader.Handler)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("failed to get contract handler: %w", err),
			EndpointID: contractReader.Handler,
		}
	}
	value, err := handler.FetchValue(ctx, contractReader.Reader, priceCache)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("failed to fetch contract value: %w", err),
			EndpointID: contractReader.Handler,
		}
	}

	defer contractReader.Reader.Close()

	fmt.Println("Contract value:", value)
	return Result{
		Value:      value,
		EndpointID: "contract:" + contractReader.Handler,
	}
}

func fetchFromRpcEndpoint(
	ctx context.Context,
	rpchandler RpcHandler,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) Result {
	handlerStr := rpchandler.Handler
	if handlerStr == "" {
		handlerStr = "generic"
	}

	handler, err := rpc_handler.GetHandler(handlerStr)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("failed to get RPC handler: %w", err),
			EndpointID: rpchandler.Handler,
		}
	}

	value, err := handler.FetchValue(ctx, rpchandler.Reader, rpchandler.Invert, rpchandler.UsdViaID, priceCache)
	if err != nil {
		return Result{
			Err:        fmt.Errorf("failed to fetch RPC value: %w", err),
			EndpointID: rpchandler.Handler,
		}
	}

	fmt.Println("RPC value:", value, rpchandler.EndpointID)
	return Result{
		Value:      value,
		EndpointID: rpchandler.Handler,
	}
}

// aggregateResults aggregates results using the specified method
func aggregateResults(results []Result, method, responseType string) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no results to aggregate")
	}

	// Extract values
	values := make([]float64, len(results))
	for i, result := range results {
		values[i] = result.Value
	}

	switch strings.ToLower(method) {
	case "median":
		return MedianInHex(values, responseType)
	// case "mode":
	// return ModeInHex(values, responseType)
	default:
		return "", fmt.Errorf("unsupported aggregation method: %s", method)
	}
}

func ConvertFloat64ToString(num float64) string {
	return strconv.FormatFloat(num, 'f', -1, 64)
}
