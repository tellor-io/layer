package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appflags "github.com/tellor-io/layer/app/flags"
	"github.com/tellor-io/layer/daemons/flags"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
)

const defaultGas = uint64(300000)

type Client struct {
	// reporter account name
	AccountName string
	// Query clients
	OracleQueryClient oracletypes.QueryClient
	StakingClient     stakingtypes.QueryClient
	ReporterClient    reportertypes.QueryClient

	cosmosCtx        client.Context
	MarketParams     []pricefeedtypes.MarketParam
	MarketToExchange *pricefeedservertypes.MarketToExchangePrices

	StakingKeeper stakingkeeper.Keeper

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
	accountName := c.AccountName
	c.cosmosCtx = c.cosmosCtx.WithChainID("layer")
	homeDir := c.GetNodeHomeDir()
	if homeDir != "" {
		c.cosmosCtx = c.cosmosCtx.WithHomeDir(homeDir)
	} else {
		panic("homeDir is empty")
	}
	fromAddr, fromName, _, err := client.GetFromFields(c.cosmosCtx, c.cosmosCtx.Keyring, accountName)
	if err != nil {
		panic(fmt.Errorf("error getting address from keyring: %v", err))
	} else {
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
	if err := client.CreateReporter(ctx, ctxGetter); err != nil {
		client.logger.Error("Error creating reporter: %w", "err", err)
		panic(err)
	}

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

// MsgCreateReporter creates a staked reporter
func (c *Client) CreateReporter(ctx context.Context, ctxGetter func(int64, bool) (sdk.Context, error)) error {
	for {
		latestHeight, err := c.LatestBlockHeight(ctx)
		if err != nil {
			c.logger.Error("Error getting latest block height: %v", err)
			panic(err)
		}

		if latestHeight < 2 {
			time.Sleep(time.Second)
			continue
		}

		appCtx, err := ctxGetter(0, false)
		if err != nil {
			c.logger.Error("Error getting context: %v", err)
			time.Sleep(time.Second * 5)
			appCtx, err = ctxGetter(0, false)
			if err != nil {
				c.logger.Error("Error getting context: %v", err)
				panic(err)
			}
		}

		validators, err := c.StakingKeeper.GetDelegatorValidators(appCtx, c.accAddr, 1)
		if err != nil {
			return err
		}
		if len(validators.Validators) == 0 {
			c.logger.Info("No validators found, waiting for validators to be available")
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	// get reporter
	appCtx, err := ctxGetter(0, false)
	if err != nil {
		return err
	}

	validators, err := c.StakingKeeper.GetDelegatorValidators(appCtx, c.accAddr, 1)
	if err != nil {
		return err
	}

	val := validators.Validators[0]

	// stake reporter transaction, reporter is determined by LAYERD_NODE_HOME environment variable
	// should make this configurable by user :time.Sleep(time.Second)todo
	// staking 1 TRB
	amtToStake := math.NewInt(1_000_000) // one TRB
	commission := reportertypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(3, 1),
		math.LegacyNewDecWithPrec(1, 1), time.Now())
	source := reportertypes.TokenOrigin{ValidatorAddress: val.GetOperator(), Amount: amtToStake}
	msgCreateReporter := &reportertypes.MsgCreateReporter{
		Reporter:     c.accAddr.String(),
		Amount:       amtToStake,
		TokenOrigins: []*reportertypes.TokenOrigin{&source},
		Commission:   &commission,
	}
	return c.sendTx(ctx, msgCreateReporter, nil)
}

func (c *Client) SubmitReport(ctx context.Context) error {
	querydata, err := c.CurrentQuery(ctx)
	if err != nil {
		return fmt.Errorf("error calling 'CurrentQuery': %v", err)
	}
	value, err := c.median(querydata)
	if err != nil {
		return fmt.Errorf("error getting median from median client': %v", err)
	}
	c.logger.Info("SubmitReport", "Median value", value)
	// Salt and hash the value
	salt, err := oracleutils.Salt(32)
	if err != nil {
		return fmt.Errorf("error generating salt: %v", err)
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
		return fmt.Errorf("error getting account number sequence for 'MsgCommitReport': %v", err)
	}
	err = c.sendTx(ctx, msgCommit, &seq)
	if err != nil {
		return fmt.Errorf("error generating 'MsgCommitReport': %v", err)
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

func (c *Client) GetNodeHomeDir() string {
	globalHome := os.ExpandEnv("$HOME/.layer")
	nodeHome := os.Getenv("LAYERD_NODE_HOME")

	if strings.HasPrefix(nodeHome, globalHome+"/") {
		return nodeHome
	}
	return ""
}
