package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/spf13/viper"
	globalfeetypes "github.com/strangelove-ventures/globalfee/x/globalfee/types"
	customquery "github.com/tellor-io/layer/daemons/custom_query"
	"github.com/tellor-io/layer/daemons/flags"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
	tokenbridgetipstypes "github.com/tellor-io/layer/daemons/server/types/token_bridge_tips"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const defaultGas = uint64(300000)

var (
	commitedIds   = make(map[uint64]bool)
	depositTipMap = make(map[uint64]bool) // map of deposit tips already sent to bridge daemon

	// Atomic counter for unordered tx timeout uniqueness (nanosecond increment)
	txTimeoutNonce uint64
)

var mutex = &sync.RWMutex{}

type TxChannelInfo struct {
	Msg         sdk.Msg
	isBridge    bool
	NumRetries  uint8
	QueryMetaId uint64 // track which queryMeta this transaction is for (0 if not applicable)
}

type Client struct {
	// reporter account name
	AccountName string
	// Query clients
	OracleQueryClient oracletypes.QueryClient

	ReporterClient  reportertypes.QueryClient
	CmtService      cmtservice.ServiceClient
	GlobalfeeClient globalfeetypes.QueryClient
	AuthClient      authtypes.QueryClient

	cosmosCtx            client.Context
	MarketParams         []pricefeedtypes.MarketParam
	MarketToExchange     *pricefeedservertypes.MarketToExchangePrices
	TokenDepositsCache   *tokenbridgetypes.DepositReports
	TokenBridgeTipsCache *tokenbridgetipstypes.DepositTips
	Custom_query         map[string]customquery.QueryConfig

	accAddr   sdk.AccAddress
	minGasFee string
	// logger is the logger for the daemon.
	logger     log.Logger
	txChan     chan TxChannelInfo
	PriceGuard *PriceGuard
}

// GetUniqueUnorderedTimeout generates a unique timeout timestamp for unordered transactions.
// Returns current time + 30 seconds + atomic nanosecond increment for uniqueness.
// https://docs.cosmos.network/v0.53/build/architecture/adr-070-unordered-account
func (c *Client) GetUniqueUnorderedTimeout() time.Time {
	// Atomically increment nonce and add to base timeout (30 seconds from now)
	nonce := atomic.AddUint64(&txTimeoutNonce, 1)
	return time.Now().Add(30 * time.Second).Add(time.Duration(nonce) * time.Nanosecond)
}

func NewClient(logger log.Logger, valGasMin string) *Client {
	logger = logger.With("module", "reporter-client")
	txChan := make(chan TxChannelInfo)
	return &Client{
		cosmosCtx: client.Context{},
		logger:    logger,
		minGasFee: valGasMin,
		txChan:    txChan,
	}
}

func (c *Client) Start(
	ctx context.Context,
	flags flags.DaemonFlags,
	grpcAddress string,
	grpcClient daemontypes.GrpcClient,
	marketParams []pricefeedtypes.MarketParam,
	marketToExchange *pricefeedservertypes.MarketToExchangePrices,
	tokenDepositsCache *tokenbridgetypes.DepositReports,
	tokenBridgeTipsCache *tokenbridgetipstypes.DepositTips,
	custom_queries map[string]customquery.QueryConfig,
	chainId string,
) error {
	// Log the daemon flags.
	c.logger.Info(
		"Starting reporter daemon with flags",
	)

	c.MarketParams = marketParams
	c.MarketToExchange = marketToExchange

	c.TokenDepositsCache = tokenDepositsCache
	c.TokenBridgeTipsCache = tokenBridgeTipsCache
	c.Custom_query = custom_queries
	// Make a connection to the Cosmos gRPC query services.
	conn, err := grpcClient.NewTcpConnection(ctx, grpcAddress)
	if err != nil {
		c.logger.Error("Failed to establish gRPC connection to Cosmos gRPC query services", "error", err)
		return err
	}
	defer func() {
		if connErr := grpcClient.CloseConnection(conn); connErr != nil {
			err = connErr
		}
	}()

	// Initialize the query clients. These are used to query the Cosmos gRPC query services.
	c.OracleQueryClient = oracletypes.NewQueryClient(conn)
	c.ReporterClient = reportertypes.NewQueryClient(conn)
	c.GlobalfeeClient = globalfeetypes.NewQueryClient(conn)
	c.CmtService = cmtservice.NewServiceClient(conn)
	c.AuthClient = authtypes.NewQueryClient(conn)

	ticker := time.NewTicker(time.Millisecond * 200)
	stop := make(chan bool)

	keyName := viper.GetString("from")
	homeDir := viper.GetString("home")
	brdcstMode := viper.GetString("broadcast-mode")
	nodeUri := viper.GetString("node")
	kb := viper.GetString("keyring-backend")

	// Read price guard config
	priceGuardEnabled := viper.GetBool("price-guard-enabled")

	var priceGuardThreshold float64
	var priceGuardMaxAge time.Duration

	if priceGuardEnabled {
		// If price guard is enabled, require explicit configuration
		if !viper.IsSet("price-guard-threshold") {
			return fmt.Errorf("price-guard-enabled is true but price-guard-threshold is not set")
		}
		priceGuardThreshold = viper.GetFloat64("price-guard-threshold")
		if priceGuardThreshold <= 0 {
			return fmt.Errorf("price-guard-threshold must be greater than 0, got: %f", priceGuardThreshold)
		}

		if !viper.IsSet("price-guard-max-age") {
			return fmt.Errorf("price-guard-enabled is true but price-guard-max-age is not set")
		}
		priceGuardMaxAge = viper.GetDuration("price-guard-max-age")
		if priceGuardMaxAge <= 0 {
			return fmt.Errorf("price-guard-max-age must be greater than 0, got: %s", priceGuardMaxAge)
		}
	} else {
		// If price guard is disabled, error if any other price guard flags are set
		if viper.IsSet("price-guard-threshold") || viper.IsSet("price-guard-max-age") {
			return fmt.Errorf("price-guard flags are set but price-guard-enabled is false")
		}
	}

	c.PriceGuard = NewPriceGuard(priceGuardThreshold, priceGuardMaxAge, priceGuardEnabled, c.logger)

	// Log price guard configuration
	if priceGuardEnabled {
		c.logger.Info("Price guard enabled",
			"threshold", fmt.Sprintf("%.5f%%", priceGuardThreshold*100),
			"max_age", priceGuardMaxAge.String(),
		)
	} else {
		c.logger.Info("Price guard disabled")
	}

	c.cosmosCtx = c.cosmosCtx.WithChainID(chainId)
	c.cosmosCtx = c.cosmosCtx.WithHomeDir(homeDir)
	c.cosmosCtx = c.cosmosCtx.WithKeyringDir(homeDir)
	c.cosmosCtx = c.cosmosCtx.WithGRPCClient(conn)
	c.cosmosCtx = c.cosmosCtx.WithBroadcastMode(brdcstMode)
	c.cosmosCtx = c.cosmosCtx.WithAccountRetriever(authtypes.AccountRetriever{})

	rpcClient, err := rpchttp.New(nodeUri, "/websocket")
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	c.cosmosCtx = c.cosmosCtx.WithClient(rpcClient)

	encodingConfig := CreateEncodingConfig()
	c.cosmosCtx = c.cosmosCtx.WithCodec(encodingConfig.Codec).WithInterfaceRegistry(encodingConfig.InterfaceRegistry).WithTxConfig(encodingConfig.TxConfig)

	kr, err := keyring.New("", kb, homeDir, nil, encodingConfig.Codec)
	if err != nil {
		return err
	}
	record, err := kr.Key(keyName)
	if err != nil {
		return err
	}
	addr, err := record.GetAddress()
	if err != nil {
		return err
	}

	c.cosmosCtx = c.cosmosCtx.WithKeyring(kr)
	c.cosmosCtx = c.cosmosCtx.WithFrom(keyName).WithFromName(keyName).WithFromAddress(addr)
	c.accAddr = c.cosmosCtx.GetFromAddress()

	StartReporterDaemonTaskLoop(
		c,
		ctx,
		flags,
		ticker,
		stop,
	)

	return nil
}

func StartReporterDaemonTaskLoop(
	client *Client,
	ctx context.Context,
	flags flags.DaemonFlags,
	ticker *time.Ticker,
	stop <-chan bool,
) {
	reporterCreated := false
	// Check if the reporter is created
	for !reporterCreated {
		reporterCreated = client.checkReporter(ctx)
		if reporterCreated {
			err := client.SetGasPrice(ctx)
			if err != nil {
				client.logger.Error("Setting gas, required before reporter can report", "error", err)
				reporterCreated = false
			}
		} else {
			time.Sleep(time.Second)
			client.logger.Warn("Checking if reporter is created", "reporterCreated", reporterCreated)
		}
	}

	time.Sleep(5 * time.Second)
	err := client.WaitForNextBlock(ctx)
	if err != nil {
		client.logger.Error("Waiting for next block", "error", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go client.BroadcastTxMsgToChain()

	wg.Add(1)
	go client.MonitorCyclelistQuery(ctx, &wg)

	wg.Add(1)
	go client.MonitorTokenBridgeReports(ctx, &wg)

	wg.Add(1)
	go client.MonitorForTippedQueries(ctx, &wg)

	wg.Add(1)
	go client.WithdrawAndStakeEarnedRewardsPeriodically(ctx, &wg)

	wg.Wait()
}

func (c *Client) checkReporter(ctx context.Context) bool {
	_, err := c.ReporterClient.SelectorReporter(ctx, &reportertypes.QuerySelectorReporterRequest{SelectorAddress: c.accAddr.String()})
	c.logger.Debug("Checking if reporter is created", "error", err)
	return err == nil
}
