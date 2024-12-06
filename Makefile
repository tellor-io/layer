#!/usr/bin/make -f

GOPATH=$(shell go env GOPATH)

COSMOS_VERSION=$(shell go list -m all | grep "github.com/cosmos/cosmos-sdk" | awk '{print $$NF}')
HTTPS_GIT := https://github.com/tellor-io/layer.git
DOCKER := $(shell which docker)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  ifeq (,$(VERSION))
    ifeq ($(shell git status --porcelain),)
    	VERSION := $(BRANCH)
    else
    	VERSION := $(BRANCH)-dirty
    endif
  endif
endif

ldflags := $(LDFLAGS)
ldflags += -X github.com/cosmos/cosmos-sdk/version.Name=Layer \
	-X github.com/cosmos/cosmos-sdk/version.AppName=layerd \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -ldflags '$(ldflags)'

###############################################################################
###                              Building / Install                         ###
###############################################################################

install: go.sum
	@echo "Installing layerd..."
	@go install -mod=readonly $(BUILD_FLAGS) ./cmd/layerd
	@echo "Completed install!"
.PHONY: install

build: mod
	@cd ./cmd/layerd
	@mkdir -p build/
	@go build $(BUILD_FLAGS) -o build/ ./cmd/layerd
.PHONY: build

mod:
	@echo "--> Updating go.mod"
	@go mod tidy
.PHONY: mod

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
golangci_version=v1.61.0

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
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage="$(DOCKER)" run -e BUF_CACHE_DIR=/tmp/buf --rm -v "$(CURDIR)":/workspace:rw --user ${CURRENT_UID}:${CURRENT_GID} --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen format

proto-gen:
	@go install cosmossdk.io/orm/cmd/protoc-gen-go-cosmos-orm@v1.0.0-beta.3
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh
	@go mod tidy

proto-format:
	@echo "Formatting Protobuf files"
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-swagger-gen:
	@./scripts/protoc-swagger-gen.sh

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

.PHONY: proto-all proto-gen proto-swagger-gen proto-format proto-lint proto-check-breaking

###############################################################################
###                                tests                                    ###
###############################################################################
test:
	@go test -v ./... -short

e2e:
	@cd e2e && go test -v -race ./... -timeout 20m

.PHONY: test e2e
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
	@go run github.com/vektra/mockery/v2 --name=BridgeKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks
	@go run github.com/vektra/mockery/v2 --name=RegistryKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks
	@go run github.com/vektra/mockery/v2 --name=ReporterKeeper --dir=$(CURDIR)/x/oracle/types --recursive --output=./x/oracle/mocks

mock-gen-registry:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/registry/types --recursive --output=./x/registry/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/registry/types --recursive --output=./x/registry/mocks
	@go run github.com/vektra/mockery/v2 --name=RegistryHooks --dir=$(CURDIR)/x/registry/types --recursive --output=./x/registry/mocks

mock-gen-reporter:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
	@go run github.com/vektra/mockery/v2 --name=OracleKeeper --dir=$(CURDIR)/x/reporter/types --recursive --output=./x/reporter/mocks
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

mock-gen-app:
	@go run github.com/vektra/mockery/v2 --name=StakingKeeper --dir=$(CURDIR)/app/ --recursive --output=./app/mocks
	@go run github.com/vektra/mockery/v2 --name=BridgeKeeper --dir=$(CURDIR)/app/ --recursive --output=./app/mocks
	@go run github.com/vektra/mockery/v2 --name=OracleKeeper --dir=$(CURDIR)/app/ --recursive --output=./app/mocks
	@go run github.com/vektra/mockery/v2 --name=Keyring --dir=$(GOPATH)/pkg/mod/github.com/cosmos/cosmos-sdk@$(COSMOS_VERSION)/crypto/keyring --recursive --output=./app/mocks

mock-gen:
	$(MAKE) mock-gen-bridge
	$(MAKE) mock-gen-dispute
	$(MAKE) mock-gen-mint
	$(MAKE) mock-gen-oracle
	$(MAKE) mock-gen-registry
	$(MAKE) mock-gen-reporter

.PHONY: mock-gen mock-gen-bridge mock-gen-dispute mock-gen-mint mock-gen-oracle mock-gen-registry mock-gen-reporter mock-gen-daemon

get-heighliner:
	git clone --depth 1 https://github.com/strangelove-ventures/heighliner.git
	cd heighliner && go install
	@sleep 0.1
	@echo ✅ heighliner installed to $(shell which heighliner)

local-image:
ifeq (,$(shell which heighliner))
	echo 'heighliner' binary not found. Consider running `make get-heighliner`
else
	heighliner build -c layer --local --dockerfile cosmos --build-target "make install" --binaries "/go/bin/layerd"
endif

get-localic:
	@echo "Installing local-interchain"
	git clone --depth 1 https://github.com/strangelove-ventures/interchaintest.git
	cd interchaintest/local-interchain && make install
	@sleep 0.1
	@echo ✅ local-interchain installed $(shell which local-ic)


local-devnet:
ifeq (,$(shell which local-ic))
	echo 'local-ic' binary not found. Consider running `make get-localic`
else
	echo "Starting local interchain"
	cd local_devnet && ICTEST_HOME=. local-ic start layer.json
	
endif
.PHONY: get-heighliner local-image get-localic local-devnet