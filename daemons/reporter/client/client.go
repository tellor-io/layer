package client

import (
	"context"
	"fmt"
	"strings"
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
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
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

	// Resources that need cleanup
	grpcConn    *grpc.ClientConn
	grpcClient  daemontypes.GrpcClient
	wg          sync.WaitGroup
	broadcastWg sync.WaitGroup // Tracks goroutines in BroadcastTxMsgToChain
	stopOnce    sync.Once
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
	c.logger.Info("Establishing gRPC connection", "address", grpcAddress)
	conn, err := grpcClient.NewTcpConnection(ctx, grpcAddress)
	if err != nil {
		c.logger.Error("Failed to establish gRPC connection to Cosmos gRPC query services", "error", err, "address", grpcAddress)
		return err
	}
	c.logger.Info("gRPC connection established successfully", "address", grpcAddress)
	// Store connection and grpcClient for cleanup
	c.grpcConn = conn
	c.grpcClient = grpcClient

	// Initialize the query clients. These are used to query the Cosmos gRPC query services.
	c.OracleQueryClient = oracletypes.NewQueryClient(conn)
	c.ReporterClient = reportertypes.NewQueryClient(conn)
	c.GlobalfeeClient = globalfeetypes.NewQueryClient(conn)
	c.CmtService = cmtservice.NewServiceClient(conn)
	c.AuthClient = authtypes.NewQueryClient(conn)

	keyName := viper.GetString("from")
	homeDir := viper.GetString("home")
	brdcstMode := viper.GetString("broadcast-mode")
	nodeUri := viper.GetString("node")
	kb := viper.GetString("keyring-backend")

	// Read price guard config
	priceGuardEnabled := viper.GetBool("price-guard-enabled")
	updateOnBlocked := viper.GetBool("price-guard-update-on-blocked")

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

		if !viper.IsSet("price-guard-update-on-blocked") {
			return fmt.Errorf("price-guard-enabled is true but price-guard-update-on-blocked is not set")
		}
	} else {
		// If price guard is disabled, error if any other price guard flags are set
		if viper.IsSet("price-guard-threshold") || viper.IsSet("price-guard-max-age") || viper.IsSet("price-guard-update-on-blocked") {
			return fmt.Errorf("price-guard flags are set but price-guard-enabled is false")
		}
	}

	c.PriceGuard = NewPriceGuard(priceGuardThreshold, priceGuardMaxAge, priceGuardEnabled, updateOnBlocked, c.logger)

	// Read auto unbonding configuration
	autoUnbondingFrequency := viper.GetUint32("auto-unbonding-frequency")
	autoUnbondingAmount := viper.GetUint32("auto-unbonding-amount")
	autoUnbondingMaxStakePercentage := viper.GetString("auto-unbonding-max-stake-percentage")

	if autoUnbondingFrequency > 0 {
		if autoUnbondingAmount == 0 {
			return fmt.Errorf("auto-unbonding-amount must be greater than 0 when auto-unbonding-frequency is set")
		}
		maxStakePercentage, err := math.LegacyNewDecFromStr(autoUnbondingMaxStakePercentage)
		if err != nil {
			return fmt.Errorf("auto-unbonding-max-stake-percentage must be a valid decimal, got: %s", autoUnbondingMaxStakePercentage)
		}
		if maxStakePercentage.LT(math.LegacyZeroDec()) || maxStakePercentage.GT(math.LegacyNewDecFromInt(math.NewInt(1))) {
			return fmt.Errorf("auto-unbonding-max-stake-percentage must be between 0.0 and 1.0, got: %s", autoUnbondingMaxStakePercentage)
		}
	}

	// Log price guard configuration
	if priceGuardEnabled {
		c.logger.Info("Price guard enabled",
			"threshold", fmt.Sprintf("%.5f%%", priceGuardThreshold*100),
			"max_age", priceGuardMaxAge.String(),
			"update_on_blocked", updateOnBlocked,
		)
	} else {
		c.logger.Info("Price guard disabled")
	}

	if autoUnbondingFrequency > 0 {
		c.logger.Info("Auto unbonding enabled",
			"frequency", autoUnbondingFrequency,
			"amount", autoUnbondingAmount,
			"max_stake_percentage", autoUnbondingMaxStakePercentage,
		)
	} else {
		c.logger.Info("Auto unbonding disabled")
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
		&c.wg,
	)

	return nil
}

func StartReporterDaemonTaskLoop(
	client *Client,
	ctx context.Context,
	flags flags.DaemonFlags,
	wg *sync.WaitGroup,
) {
	reporterCreated := false
	// Check if the reporter is created
	for !reporterCreated {
		reporterCreated = client.checkReporter(ctx)
		if reporterCreated {
			client.logger.Info("Reporter exists, setting gas price")
			err := client.SetGasPrice(ctx)
			if err != nil {
				client.logger.Error("Setting gas price failed, required before reporter can report", "error", err)
				reporterCreated = false
				time.Sleep(time.Second)
			} else {
				client.logger.Info("Gas price set successfully", "gas_price", client.minGasFee)
			}
		} else {
			time.Sleep(time.Second)
			client.logger.Warn("Reporter not found, retrying...", "selector_address", client.accAddr.String())
		}
	}

	time.Sleep(5 * time.Second)
	err := client.WaitForNextBlock(ctx)
	if err != nil {
		client.logger.Error("Waiting for next block", "error", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		client.BroadcastTxMsgToChain(ctx)
	}()

	wg.Add(1)
	go client.MonitorCyclelistQuery(ctx, wg)

	wg.Add(1)
	go client.MonitorTokenBridgeReports(ctx, wg)

	wg.Add(1)
	go client.MonitorForTippedQueries(ctx, wg)

	wg.Add(1)
	go client.WithdrawAndStakeEarnedRewardsPeriodically(ctx, wg)

	wg.Add(1)
	go client.AutoUnbondStakePeriodically(ctx, wg)

	wg.Wait()
}

func (c *Client) checkReporter(ctx context.Context) bool {
	c.logger.Info("Checking if reporter is created", "address", c.accAddr.String())

	// Retry logic for connection issues - gRPC connections are lazy and may fail initially
	maxRetries := 3
	retryDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Info("Retrying reporter check", "attempt", attempt+1, "max_retries", maxRetries)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}

		// First try to check if the address is a reporter directly
		reporterResp, err := c.ReporterClient.Reporter(ctx, &reportertypes.QueryReporterRequest{ReporterAddress: c.accAddr.String()})
		if err == nil {
			c.logger.Info("Reporter found (direct)", "address", c.accAddr.String(), "reporter", reporterResp)
			return true
		}

		// Check if it's a connection error that we should retry
		if isConnectionError(err) && attempt < maxRetries-1 {
			c.logger.Debug("Connection error, will retry", "error", err, "attempt", attempt+1)
			continue
		}

		c.logger.Debug("Direct reporter check failed, trying selector", "error", err, "address", c.accAddr.String())
		// If not a reporter, check if it's a selector that has selected a reporter
		selectorResp, err := c.ReporterClient.SelectorReporter(ctx, &reportertypes.QuerySelectorReporterRequest{SelectorAddress: c.accAddr.String()})
		if err == nil {
			c.logger.Info("Reporter found (via selector)", "address", c.accAddr.String(), "reporter", selectorResp.Reporter)
			return true
		}

		// Check if it's a connection error that we should retry
		if isConnectionError(err) && attempt < maxRetries-1 {
			c.logger.Debug("Connection error on selector check, will retry", "error", err, "attempt", attempt+1)
			continue
		}

		// If we get here and it's not a connection error, or we've exhausted retries, return false
		if !isConnectionError(err) || attempt == maxRetries-1 {
			c.logger.Info("Reporter check failed - address is neither a reporter nor a selector", "error", err, "address", c.accAddr.String())
			return false
		}
	}

	return false
}

// isConnectionError checks if an error is a transient connection error that should be retried
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "transport: Error while dialing") ||
		strings.Contains(errStr, "Unavailable")
}

// Stop stops the reporter client gracefully
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		// Close the transaction channel to signal BroadcastTxMsgToChain to stop
		close(c.txChan)

		// Wait for all goroutines to finish
		c.wg.Wait()

		// Wait for broadcast goroutines to finish
		c.broadcastWg.Wait()

		// Close gRPC connection
		if c.grpcConn != nil && c.grpcClient != nil {
			if err := c.grpcClient.CloseConnection(c.grpcConn); err != nil {
				c.logger.Error("Failed to close gRPC connection", "error", err)
			}
		}
	})
}
