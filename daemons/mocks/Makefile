GOPATH=$(shell go env GOPATH)

COSMOS_VERSION=$(shell go list -m all | grep "github.com/cosmos/cosmos-sdk" | awk '{print $$NF}')
COSMOS_LOG_VERSION=$(shell go list -m all | grep "cosmossdk.io/log" | awk '{print $$NF}')

mock-clean:
	@rm -f ./*.go

mock-gen:
	@go run github.com/vektra/mockery/v2 --name=AppOptions --dir=$(GOPATH)/pkg/mod/github.com/cosmos/cosmos-sdk@$(COSMOS_VERSION)/server/types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=ExchangeQueryHandler --dir=../pricefeed/client/queryhandler --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=ExchangeToMarketPrices --dir=../pricefeed/client/types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=GrpcClient --dir=../types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=Logger --dir=$(GOPATH)/pkg/mod/cosmossdk.io/log@$(COSMOS_LOG_VERSION) --filename=logger.go --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=PricefeedMutableMarketConfigs --dir=../pricefeed/client/types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=QueryClient --dir=../../testutil/grpc --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=RequestHandler --dir=../types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=TimeProvider --dir=../../lib/time --recursive --output=.