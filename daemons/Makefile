#!/usr/bin/make -f

# Fix version extraction and use hardcoded values for problematic dependencies
GOPATH=$(shell go env GOPATH)
COSMOS_VERSION=$(shell go list -m all | grep "github.com/cosmos/cosmos-sdk" | awk '{print $$NF}' | head -1)
COSMOS_LOG_VERSION=$(shell go list -m all | grep "cosmossdk.io/log" | awk '{print $$NF}' | head -1)
# Use a known stable version instead of trying to extract it
GOGO_PROTO_VERSION=v1.3.2
GRPC_GATEWAY_VERSION=v1.16.0

.PHONY: install build clean mod

install: go.sum
	@echo "Installing reporterd..."
	@go install -mod=readonly ./cmd
	@echo "Completed install!"

build:
	@echo "Building reporterd..."
	go build -o bin/reporterd ./cmd/main.go

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

mod:
	@echo "--> Updating go.mod"
	@go mod tidy


###############################################################################
###                                tests                                    ###
###############################################################################
.PHONY: test mock-gen-daemon

mock-gen-daemon:
	@go run github.com/vektra/mockery/v2 --name=AppOptions --dir=$(GOPATH)/pkg/mod/github.com/cosmos/cosmos-sdk@$(COSMOS_VERSION)/server/types --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=ExchangeQueryHandler --dir=$(CURDIR)/pricefeed/client/queryhandler --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=ExchangeToMarketPrices --dir=$(CURDIR)/pricefeed/client/types --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=GrpcClient --dir=$(CURDIR)/types --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=Logger --dir=$(GOPATH)/pkg/mod/cosmossdk.io/log@$(COSMOS_LOG_VERSION) --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=PricefeedMutableMarketConfigs --dir=$(CURDIR)/pricefeed/client/types --filename=PriceFeedMutableMarketConfigs.go --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=QueryClient --dir=$(CURDIR)/testutil/grpc --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=RequestHandler --dir=$(CURDIR)/types --recursive --output=$(CURDIR)/mocks
	@go run github.com/vektra/mockery/v2 --name=TimeProvider --dir=$(CURDIR)/lib/time --recursive --output=$(CURDIR)/mocks


test:
	@echo "Running tests..."
	go test -v ./...

###############################################################################
###                                linting                                  ###
###############################################################################
.PHONY: lint lint-fix install-lint-deps

lint:
	@echo "Linting daemon code..."
	golangci-lint run ./...

# Fix linting issues where possible
lint-fix:
	@echo "Fixing linting issues..."
	golangci-lint run --fix ./...

install-lint-deps:
	@echo "Checking for golangci-lint..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	else \
		echo "golangci-lint already installed"; \
	fi


###############################################################################
###                                 proto                                   ###
###############################################################################
.PHONY: proto proto-gen proto-install proto-check proto-deps

proto-gen:
	@echo "Generating protocol buffer files using buf..."
	@if ! command -v buf > /dev/null; then \
		echo "buf not found. Installing..."; \
		go install github.com/bufbuild/buf/cmd/buf@latest; \
	fi
	@echo "Resolving dependencies..."
	@cd proto && buf dep update
	@echo "Generating code..."
	@cd proto && buf generate --template buf.gen.gogo.yaml
	@echo "Proto generation completed"

proto-install:
	@echo "Installing protobuf dependencies for Cosmos SDK compatibility..."
	@go install github.com/cosmos/gogoproto/protoc-gen-gocosmos@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@$(GRPC_GATEWAY_VERSION)
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "Proto tooling installation completed"

proto-check:
	@if ! command -v buf > /dev/null; then \
		echo "Error: buf is not installed. Please install buf first."; \
		echo "Run: go install github.com/bufbuild/buf/cmd/buf@latest"; \
		exit 1; \
	fi
	@if [ ! -d proto/daemons ]; then \
		echo "Error: proto/daemons directory does not exist."; \
		exit 1; \
	fi

# Main proto target that depends on proto-gen
proto: proto-check
	@echo "Running proto generation..."
	@$(MAKE) proto-gen
	@echo "Protocol buffers generated successfully"

proto-deps:
	@echo "Installing protobuf dependencies..."
	@$(MAKE) proto-install
