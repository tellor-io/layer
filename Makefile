#!/usr/bin/make -f

GOPATH=$(shell go env GOPATH)

COSMOS_VERSION=$(shell go list -m all | grep "github.com/cosmos/cosmos-sdk" | awk '{print $$NF}')
HTTPS_GIT := https://github.com/tellor-io/layer.git
DOCKER := $(shell which docker)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/layerd-linux-amd64 ./cmd/layerd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./build/layerd-linux-arm64 ./cmd/layerd

do-checksum-linux:
	cd build && shasum -a 256 \
    	layerd-linux-amd64 layerd-linux-arm64 \
    	> layerd-checksum-linux

build-linux-with-checksum: build-linux do-checksum-linux

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./build/layerd-darwin-amd64 ./cmd/layerd
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./build/layerd-darwin-arm64 ./cmd/layerd

build-all: build-linux build-darwin

do-checksum-darwin:
	cd build && shasum -a 256 \
		layerd-darwin-amd64 layerd-darwin-arm64 \
		> layer-checksum-darwin

build-darwin-with-checksum: build-darwin do-checksum-darwin

build-with-checksum: build-linux-with-checksum build-darwin-with-checksum

###############################################################################
###                                Linting                                  ###
###############################################################################
# Golangci-lint version
golangci_version=v1.56.2

#? setup-pre-commit: Set pre-commit git hook
setup-pre-commit:
	@cp .git/hooks/pre-commit .git/hooks/pre-commit.bak 2>/dev/null || true
	@echo "Installing pre-commit hook..."
	@ln -sf ../../scripts/hooks/pre-commit.sh .git/hooks/pre-commit
	@echo "Pre-commit hook installed successfully"

#? lint-install: Install golangci-lint
lint-install:
	@echo "--> Installing golangci-lint $(golangci_version)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)

#? lint: Run golangci-lint
lint:
	@echo "--> Running linter"
	$(MAKE) lint-install
	@./scripts/go-lint-all.bash --timeout=15m

#? lint: Run golangci-lint and fix 
lint-fix:
	@echo "--> Running linter"
	$(MAKE) lint-install
	@./scripts/go-lint-all.bash --fix

# Lint specific folders
lint-folder-fix:
	@echo "--> Running linter for specified folders: $(FOLDER)"
	$(MAKE) lint-install
	@./scripts/go-lint-all.bash $(FOLDER) --fix

.PHONY: lint lint-fix lint-folder


###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

#? proto-all: Run make proto-format proto-lint proto-gen
proto-all: proto-format proto-lint proto-gen

#? proto-gen: Generate Protobuf files
proto-gen:
	@$(protoImage) sh ./scripts/protocgen.sh

#? proto-swagger-gen: Generate Protobuf Swagger
proto-swagger-gen:
	@echo "Generating Protobuf Swagger"
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh

#? proto-format: Format proto file
proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

#? proto-lint: Lint proto file
proto-lint:
	@$(protoImage) buf lint --error-format=json

#? proto-check-breaking: Check proto file is breaking
proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

#? proto-update-deps: Update protobuf dependencies
proto-update-deps:
	@echo "Updating Protobuf dependencies"
	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(protoImageName) buf mod update

.PHONY: proto-all proto-gen proto-swagger-gen proto-format proto-lint proto-check-breaking proto-update-deps

###############################################################################
###                                MOCKS                                    ###
###############################################################################
mock-clean-all:
	@find . -type f -name "*.go" -path "*/mocks/*" -exec rm -f {} +

mock-gen-bridge:
	@go run github.com/vektra/mockery/v2 --name=StakingKeeper --dir=$(CURDIR)/x/bridge/types --recursive --output=./x/bridge/mocks
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/bridge/types --recursive --output=./x/bridge/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/bridge/types --recursive --output=./x/bridge/mocks
	@go run github.com/vektra/mockery/v2 --name=OracleKeeper --dir=$(CURDIR)/x/bridge/types --recursive --output=./x/bridge/mocks
	@go run github.com/vektra/mockery/v2 --name=ReporterKeeper --dir=$(CURDIR)/x/bridge/types --recursive --output=./x/bridge/mocks

mock-gen-dispute:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/dispute/types --recursive --output=./x/dispute/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/dispute/types --recursive --output=./x/dispute/mocks
	@go run github.com/vektra/mockery/v2 --name=OracleKeeper --dir=$(CURDIR)/x/dispute/types --recursive --output=./x/dispute/mocks
	@go run github.com/vektra/mockery/v2 --name=ReporterKeeper --dir=$(CURDIR)/x/dispute/types --recursive --output=./x/dispute/mocks

mock-gen-mint:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/mint/types --recursive --output=./x/mint/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/mint/types --recursive --output=./x/mint/mocks

mock-gen-oracle:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks
	@go run github.com/vektra/mockery/v2 --name=RegistryKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks
	@go run github.com/vektra/mockery/v2 --name=ReporterKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks

mock-gen-registry:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/registry/types --recursive --output=./x/registry/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/registry/types --recursive --output=./x/registry/mocks
	@go run github.com/vektra/mockery/v2 --name=RegistryHooks --dir=$(CURDIR)/x/registry/types --recursive --output=./x/registry/mocks

mock-gen-reporter:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
	@go run github.com/vektra/mockery/v2 --name=StakingKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
	@go run github.com/vektra/mockery/v2 --name=StakingHooks --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
	@go run github.com/vektra/mockery/v2 --name=RegistryKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks

COSMOS_LOG_VERSION=$(shell go list -m all | grep "cosmossdk.io/log" | awk '{print $$NF}')

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
mock-gen:
	$(MAKE) mock-gen-bridge
	$(MAKE) mock-gen-dispute
	$(MAKE) mock-gen-mint
	$(MAKE) mock-gen-oracle
	$(MAKE) mock-gen-registry
	$(MAKE) mock-gen-reporter

.PHONY: mock-gen mock-gen-bridge mock-gen-dispute mock-gen-mint mock-gen-oracle mock-gen-registry mock-gen-reporter mock-gen-daemon