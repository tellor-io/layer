package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/viper"
	globalfeetypes "github.com/strangelove-ventures/globalfee/x/globalfee/types"
	appflags "github.com/tellor-io/layer/app/flags"
	"github.com/tellor-io/layer/daemons/flags"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const defaultGas = uint64(300000)

var (
	commitedIds      = make(map[uint64]bool)
	depositReportMap = make(map[string]bool)
)

var mutex = &sync.RWMutex{}

type Client struct {
	// reporter account name
	AccountName string
	// Query clients
	OracleQueryClient oracletypes.QueryClient
	StakingClient     stakingtypes.QueryClient
	ReporterClient    reportertypes.QueryClient
	GlobalfeeClient   globalfeetypes.QueryClient

	cosmosCtx          client.Context
	MarketParams       []pricefeedtypes.MarketParam
	MarketToExchange   *pricefeedservertypes.MarketToExchangePrices
	TokenDepositsCache *tokenbridgetypes.DepositReports
	StakingKeeper      stakingkeeper.Keeper

	accAddr   sdk.AccAddress
	minGasFee string
	// logger is the logger for the daemon.
	logger log.Logger
}

func NewClient(clctx client.Context, logger log.Logger, accountName, valGasMin string) *Client {
	logger = logger.With("module", "reporter-client")
	return &Client{
		AccountName: accountName,
		cosmosCtx:   clctx,
		logger:      logger,
		minGasFee:   valGasMin,
	}
}

func (c *Client) Start(
	ctx context.Context,
	flags flags.DaemonFlags,
	appFlags appflags.Flags,
	grpcClient daemontypes.GrpcClient,
	marketParams []pricefeedtypes.MarketParam,
	marketToExchange *pricefeedservertypes.MarketToExchangePrices,
	tokenDepositsCache *tokenbridgetypes.DepositReports,
	// ctxGetter func(int64, bool) (sdk.Context, error),
	stakingKeeper stakingkeeper.Keeper,
	chainId string,
) error {
	// Log the daemon flags.
	c.logger.Info(
		"Starting reporter daemon with flags",
		"ReportersFlags", flags.Reporter,
	)

	c.MarketParams = marketParams
	c.MarketToExchange = marketToExchange
	c.StakingKeeper = stakingKeeper

	c.TokenDepositsCache = tokenDepositsCache

	// Make a connection to the Cosmos gRPC query services.
	queryConn, err := grpcClient.NewTcpConnection(ctx, appFlags.GrpcAddress)
	if err != nil {
		c.logger.Error("Failed to establish gRPC connection to Cosmos gRPC query services", "error", err)
		return err
	}
	defer func() {
		if connErr := grpcClient.CloseConnection(queryConn); connErr != nil {
			err = connErr
		}
	}()

	// Initialize the query clients. These are used to query the Cosmos gRPC query services.
	c.OracleQueryClient = oracletypes.NewQueryClient(queryConn)
	c.StakingClient = stakingtypes.NewQueryClient(queryConn)
	c.ReporterClient = reportertypes.NewQueryClient(queryConn)
	c.GlobalfeeClient = globalfeetypes.NewQueryClient(queryConn)

	ticker := time.NewTicker(time.Millisecond * 200)
	stop := make(chan bool)

	// get account
	c.AccountName = viper.GetString("key-name")
	if c.AccountName == "" {
		panic("account name is empty, please use --key-name flag")
	}
	accountName := c.AccountName
	c.cosmosCtx = c.cosmosCtx.WithChainID(chainId)
	homeDir := viper.GetString("home")
	if homeDir != "" {
		c.cosmosCtx = c.cosmosCtx.WithHomeDir(homeDir)
	} else {
		panic("homeDir is empty, please use --home flag")
	}
	fromAddr, fromName, _, err := client.GetFromFields(c.cosmosCtx, c.cosmosCtx.Keyring, accountName)
	if err != nil {
		panic(fmt.Errorf("error getting address from keyring: %w : Keyring Type info: %v", err, c.cosmosCtx.Keyring))
	}
	c.cosmosCtx = c.cosmosCtx.WithFrom(accountName).WithFromAddress(fromAddr).WithFromName(fromName)
	c.accAddr = c.cosmosCtx.GetFromAddress()

	StartReporterDaemonTaskLoop(
		c,
		ctx,
		flags,
		ticker,
		stop,
		// ctxGetter,
	)

	return nil
}

func StartReporterDaemonTaskLoop(
	client *Client,
	ctx context.Context,
	flags flags.DaemonFlags,
	ticker *time.Ticker,
	stop <-chan bool,
	// ctxGetter func(int64, bool) (sdk.Context, error),
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
		}
	}

	time.Sleep(5 * time.Second)
	err := client.WaitForNextBlock(ctx)
	if err != nil {
		client.logger.Error("Waiting for next block", "error", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go client.MonitorCyclelistQuery(ctx, &wg)

	wg.Add(1)
	go client.MonitorTokenBridgeReports(ctx, &wg)

	wg.Add(1)
	go client.MonitorForTippedQueries(ctx, &wg)

	wg.Wait()
}

func (c *Client) checkReporter(ctx context.Context) bool {
	_, err := c.ReporterClient.SelectorReporter(ctx, &reportertypes.QuerySelectorReporterRequest{SelectorAddress: c.accAddr.String()})
	return err == nil
}
