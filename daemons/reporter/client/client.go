package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	appflags "github.com/tellor-io/layer/app/flags"
	"github.com/tellor-io/layer/daemons/flags"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	cmttypes "github.com/cometbft/cometbft/rpc/core/types"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	helper "github.com/tellor-io/layer/lib/prices"
	"github.com/tellor-io/layer/utils"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
)

const defaultGas = uint64(300000)

type Client struct {
	// reporter account name
	AccountName string
	// Query clients
	OracleQueryClient oracletypes.QueryClient
	cosmosCtx         client.Context
	MarketParams      []pricefeedtypes.MarketParam
	MarketToExchange  *pricefeedservertypes.MarketToExchangePrices
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
) error {
	// Log the daemon flags.
	c.logger.Info(
		"Starting reporter daemon with flags",
		"ReportersFlags", flags.Reporter,
	)
	c.MarketParams = marketParams
	c.MarketToExchange = marketToExchange

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

	ticker := time.NewTicker(time.Second)
	stop := make(chan bool)

	s := &SubTaskRunnerImpl{}
	StartReporterDaemonTaskLoop(
		c,
		ctx,
		c.cosmosCtx,
		s,
		flags,
		ticker,
		stop,
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
	accountName := c.AccountName
	c.cosmosCtx = c.cosmosCtx.WithChainID("layer")
	fromAddr, fromName, _, err := client.GetFromFields(c.cosmosCtx, c.cosmosCtx.Keyring, accountName)
	if err != nil {
		panic(fmt.Errorf("error getting address from keyring: %v", err))
	}
	c.cosmosCtx = c.cosmosCtx.WithFrom(accountName).WithFromAddress(fromAddr).WithFromName(fromName)
	accAddr := c.cosmosCtx.GetFromAddress()
	c.logger.Info("SubmitReport", "Account address to sign messages", accAddr.String())

	response, err := c.OracleQueryClient.CurrentCyclelistQuery(ctx, &oracletypes.QueryCurrentCyclelistQueryRequest{})
	if err != nil {
		return fmt.Errorf("error calling 'CurrentCyclelistQuery': %v", err)
	}
	qid, err := utils.QueryIDFromDataString(response.Querydata)
	if err != nil {
		c.logger.Error("error getting query id from data string: %v", err)
	}
	c.logger.Info("SubmitReport", "next query id in cycle list", hex.EncodeToString(qid))
	// needed to wait because it kept missing the query
	time.Sleep(2 * time.Second)
	value, err := c.median(ctx, strings.ToLower(response.Querydata))
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
		Creator:   accAddr.String(),
		QueryData: response.Querydata,
		Hash:      hash,
	}
	txf := newFactory(c.cosmosCtx)

	_, seq, err := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, accAddr)
	if err != nil {
		return fmt.Errorf("error getting account number sequence for 'MsgCommitReport': %v", err)
	}
	txf = txf.WithSequence(seq)
	txf, err = txf.Prepare(c.cosmosCtx)
	if err != nil {
		return fmt.Errorf("error preparing transaction during 'MsgCommitReport': %v", err)
	}

	txn, err := txf.BuildUnsignedTx(msgCommit)
	if err != nil {
		return fmt.Errorf("error building 'MsgCommitReport' unsigned transaction: %v", err)
	}
	if err = tx.Sign(c.cosmosCtx.CmdContext, txf, c.cosmosCtx.FromName, txn, true); err != nil {
		return fmt.Errorf("error when signing 'MsgCommitReport' transaction: %v", err)
	}

	txBytes, err := c.cosmosCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return fmt.Errorf("error encoding 'MsgCommitReport' transaction: %v", err)
	}
	res, err := c.cosmosCtx.BroadcastTx(txBytes)
	if err := handleBroadcastResult(res, err); err != nil {
		return fmt.Errorf("error broadcasting 'MsgCommitReport' transaction after 'handleBroadcastResult': %v", err)
	}
	txnResult, err := c.WaitForTx(ctx, res.TxHash)
	if err != nil {
		return fmt.Errorf("error waiting for 'MsgCommitReport' transaction: %v", err)
	}
	c.logger.Info("CommitReportTxResult", "TxResult", txnResult)

	// ***********************MsgSubmitValue***************************
	msgSubmit := &oracletypes.MsgSubmitValue{
		Creator:   accAddr.String(),
		QueryData: response.Querydata,
		Value:     value,
		Salt:      salt,
	}
	// increment sequence by 1 for next transaction
	seq += 1
	txf = txf.WithSequence(seq)
	txf, err = txf.Prepare(c.cosmosCtx)
	if err != nil {
		return fmt.Errorf("error preparing transaction during 'MsgCommitReport': %v", err)
	}

	txn, err = txf.BuildUnsignedTx(msgSubmit)
	if err != nil {
		return fmt.Errorf("error building 'MsgSubmitValue' unsigned transaction: %v", err)
	}
	if err = tx.Sign(c.cosmosCtx.CmdContext, txf, c.cosmosCtx.FromName, txn, true); err != nil {
		return fmt.Errorf("error when signing 'MsgSubmitValue' transaction: %v", err)
	}
	txBytes, err = c.cosmosCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return fmt.Errorf("error encoding 'MsgSubmitValue' transaction: %v", err)
	}
	// broadcast to a CometBFT node
	res, err = c.cosmosCtx.BroadcastTx(txBytes)
	if err := handleBroadcastResult(res, err); err != nil {
		return fmt.Errorf("error broadcasting 'MsgSubmitValue' transaction after 'handleBroadcastResult': %v", err)
	}
	txnResult, err = c.WaitForTx(ctx, res.TxHash)
	if err != nil {
		return fmt.Errorf("error waiting for 'MsgSubmitValue' transaction: %v", err)
	}
	c.logger.Info("SubmitValueTxResult", "TxResult", txnResult)
	return nil
}

func newFactory(clientCtx client.Context) tx.Factory {
	return tx.Factory{}.
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		WithGasAdjustment(1.1).
		WithGas(defaultGas).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithTxConfig(clientCtx.TxConfig)
}

func (c *Client) median(ctx context.Context, querydata string) (string, error) {
	params := c.MarketParams
	prices := c.MarketToExchange
	mapping := prices.GetValidMedianPrices(params, time.Now())
	fmt.Println("Mapping:", mapping)

	mapQueryDataToMarketParams := make(map[string]pricefeedtypes.MarketParam)
	for _, marketParam := range c.MarketParams {
		mapQueryDataToMarketParams[strings.ToLower(marketParam.QueryData)] = marketParam
	}
	mp, found := mapQueryDataToMarketParams[querydata]
	if !found {
		return "", fmt.Errorf("no market param found for query data: %s", querydata)
	}
	mv := c.MarketToExchange.GetValidMedianPrices([]pricefeedtypes.MarketParam{mp}, time.Now())
	val, found := mv[mp.Id]
	if !found {
		return "", fmt.Errorf("no median values found for query data: %s", querydata)
	}

	value, err := helper.EncodePrice(float64(val), mp.Exponent)
	if err != nil {
		return "", err
	}
	return value, nil
}

func handleBroadcastResult(resp *sdk.TxResponse, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("make sure that your account has enough balance")
		}
		return err
	}

	if resp.Code > 0 {
		return fmt.Errorf("error code: '%d' msg: '%s'", resp.Code, resp.RawLog)
	}
	return nil
}
func (c *Client) WaitForTx(ctx context.Context, hash string) (*cmttypes.ResultTx, error) {
	bz, err := hex.DecodeString(hash)
	if err != nil {
		return nil, fmt.Errorf("unable to decode tx hash '%s'; err: %v", hash, err)
	}
	for {
		resp, err := c.cosmosCtx.Client.Tx(ctx, bz, false)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				// Tx not found, wait for next block and try again
				err := c.WaitForNextBlock(ctx)
				if err != nil {
					return nil, fmt.Errorf("waiting for next block: err: %v", err)
				}
				continue
			}
			return nil, fmt.Errorf("fetching tx '%s'; err: %v", hash, err)
		}
		// Tx found
		return resp, nil
	}
}

func (c Client) WaitForNextBlock(ctx context.Context) error {
	return c.WaitForNBlocks(ctx, 1)
}
func (c Client) WaitForNBlocks(ctx context.Context, n int64) error {
	start, err := c.LatestBlockHeight(ctx)
	if err != nil {
		return err
	}
	return c.WaitForBlockHeight(ctx, start+n)
}
func (c Client) LatestBlockHeight(ctx context.Context) (int64, error) {
	resp, err := c.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resp.SyncInfo.LatestBlockHeight, nil
}

func (c Client) Status(ctx context.Context) (*cmttypes.ResultStatus, error) {
	return c.cosmosCtx.Client.Status(ctx)
}
func (c Client) WaitForBlockHeight(ctx context.Context, h int64) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		latestHeight, err := c.LatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		if latestHeight >= h {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout exceeded waiting for block, err: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
