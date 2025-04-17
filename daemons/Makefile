GOPATH=$(shell go env GOPATH)
COSMOS_VERSION=$(shell go list -m all | grep "github.com/cosmos/cosmos-sdk" | awk '{print $$NF}')
COSMOS_LOG_VERSION=$(shell go list -m all | grep "cosmossdk.io/log" | awk '{print $$NF}')

build:
	@echo "Building reporterd binary..."
	@go build -o reporterd ./cmd
	@echo "Completed build!"
.PHONY: build

install: go.sum
	@echo "Installing reporterd..."
	@go install -mod=readonly ./cmd
	@echo "Completed install!"
.PHONY: install

mod:
	@echo "--> Updating go.mod"
	@go mod tidy
.PHONY: mod


###############################################################################
###                                tests                                    ###
###############################################################################


mock-gen-daemon:
	@go run github.com/vektra/mockery/v2 --name=AppOptions --dir=$(GOPATH)/pkg/mod/github.com/cosmos/cosmos-sdk@$(COSMOS_VERSION)/server/types --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=ExchangeQueryHandler --dir=$(CURDIR)/daemons/pricefeed/client/queryhandler --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=ExchangeToMarketPrices --dir=$(CURDIR)/daemons/pricefeed/client/types --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=GrpcClient --dir=$(CURDIR)/daemons/types --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=Logger --dir=$(GOPATH)/pkg/mod/cosmossdk.io/log@$(COSMOS_LOG_VERSION) --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=PricefeedMutableMarketConfigs --dir=$(CURDIR)/daemons/pricefeed/client/types --filename=PriceFeedMutableMarketConfigs.go --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=QueryClient --dir=$(CURDIR)/testutil/grpc --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=RequestHandler --dir=$(CURDIR)/daemons/types --recursive --output=$(CURDIR)/daemons/mocks
	@go run github.com/vektra/mockery/v2 --name=TimeProvider --dir=$(CURDIR)/lib/time --recursive --output=$(CURDIR)/daemons/mocks
.PHONY: mock-gen-daemon