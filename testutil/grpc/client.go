package grpc

import (
	pricefeedtypes "github.com/tellor-io/layer/daemons/server/types"
)

// QueryClient combines all the query clients used in testing into a single mock interface for testing convenience.
type QueryClient interface {
	pricefeedtypes.PriceFeedServiceClient
}
