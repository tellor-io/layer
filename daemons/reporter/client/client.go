package client

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	appflags "github.com/tellor-io/layer/app/flags"
	"github.com/tellor-io/layer/daemons/flags"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const defaultGas = uint64(300000)

type Client struct {
	// reporter account name
	AccountName string
	// Query clients
	OracleQueryClient oracletypes.QueryClient
	StakingClient     stakingtypes.QueryClient
	ReporterClient    reportertypes.QueryClient

	cosmosCtx          client.Context
	MarketParams       []pricefeedtypes.MarketParam
	MarketToExchange   *pricefeedservertypes.MarketToExchangePrices
	TokenDepositsCache *tokenbridgetypes.DepositReports
	StakingKeeper      stakingkeeper.Keeper

	accAddr sdk.AccAddress
	// logger is the logger for the daemon.
	logger log.Logger
}

func NewClient(clctx client.Context, logger log.Logger, accountName string) *Client {
	return &Client{
		AccountName: accountName,
		cosmosCtx:   clctx,
		logger:      logger,
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
	ctxGetter func(int64, bool) (sdk.Context, error),
	stakingKeeper stakingkeeper.Keeper,
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

	ticker := time.NewTicker(time.Second)
	stop := make(chan bool)

	// get account
	c.AccountName = viper.GetString("key-name")
	if c.AccountName == "" {
		panic("account name is empty, please use --key-name flag")
	}
	accountName := c.AccountName
	c.cosmosCtx = c.cosmosCtx.WithChainID("layer")
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
		c.cosmosCtx,
		&SubTaskRunnerImpl{},
		flags,
		ticker,
		stop,
		ctxGetter,
	)

	return nil
}

func StartReporterDaemonTaskLoop(
	client *Client,
	ctx context.Context,
	cosmosClient client.Context,
	s SubTaskRunner,
	flags flags.DaemonFlags,
	ticker *time.Ticker,
	stop <-chan bool,
	ctxGetter func(int64, bool) (sdk.Context, error),
) {
	for {
		select {
		case <-ticker.C:
			if err := s.RunReporterDaemonTaskLoop(
				ctx,
				client,
				cosmosClient,
			); err != nil {
				client.logger.Error("Reporter daemon returned error", "error", err)
			} else {
				client.logger.Info("Reporter daemon task completed successfully")
			}
		case <-stop:
			return
		}
	}
}

func (c *Client) SubmitReport(ctx context.Context) error {
	querydata, value, err := c.deposits()
	if err != nil {
		querydata, err = c.CurrentQuery(ctx)
		if err != nil {
			return fmt.Errorf("error calling 'CurrentQuery': %w", err)
		}
		value, err = c.median(querydata)
		if err != nil {
			return fmt.Errorf("error getting median from median client': %w", err)
		}
	} else {
		// delete this
		c.logger.Info("Submitting for token bridge")
	}
	c.logger.Info("SubmitReport", "Median value", value)
	// Salt and hash the value
	salt, err := oracleutils.Salt(32)
	if err != nil {
		return fmt.Errorf("error generating salt: %w", err)
	}
	hash := oracleutils.CalculateCommitment(value, salt)

	// ***********************MsgCommitReport***************************
	msgCommit := &oracletypes.MsgCommitReport{
		Creator:   c.accAddr.String(),
		QueryData: querydata,
		Hash:      hash,
	}

	_, seq, err := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, c.accAddr)
	if err != nil {
		return fmt.Errorf("error getting account number sequence for 'MsgCommitReport': %w", err)
	}
	err = c.sendTx(ctx, msgCommit, &seq)
	if err != nil {
		return fmt.Errorf("error generating 'MsgCommitReport': %w", err)
	}

	// ***********************MsgSubmitValue***************************
	msgSubmit := &oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: querydata,
		Value:     value,
		Salt:      salt,
	}
	// no need to call GetAccountNumberSequence here, just increment sequence by 1 for next transaction
	seq++
	return c.sendTx(ctx, msgSubmit, &seq)
}
