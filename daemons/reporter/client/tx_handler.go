package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	cmttypes "github.com/cometbft/cometbft/rpc/core/types"
	globalfeetypes "github.com/strangelove-ventures/globalfee/x/globalfee/types"
	"github.com/tellor-io/layer/daemons/lib/metrics"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

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
	waiting := true
	bz, err := hex.DecodeString(hash)
	if err != nil {
		return nil, fmt.Errorf("unable to decode tx hash '%s'; err: %w", hash, err)
	}

	waitedBlockCount := 0
	for waiting {
		resp, err := c.cosmosCtx.Client.Tx(ctx, bz, false)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				if waitedBlockCount == 2 {
					return nil, fmt.Errorf("waited for next block and transaction is still not found")
				}
				err := c.WaitForNextBlock(ctx)
				if err != nil {
					return nil, fmt.Errorf("waiting for next block: err: %w", err)
				}
				waitedBlockCount++
				continue
			}
			return nil, fmt.Errorf("fetching tx '%s'; err: %w", hash, err)
		}
		// Tx found
		return resp, nil
	}
	return nil, fmt.Errorf("fetching tx '%s'; err: %w", hash, err)
}

func (c *Client) WaitForNextBlock(ctx context.Context) error {
	return c.WaitForNBlocks(ctx, 1)
}

func (c *Client) WaitForNBlocks(ctx context.Context, n int64) error {
	start, err := c.LatestBlockHeight(ctx)
	if err != nil {
		return err
	}
	return c.WaitForBlockHeight(ctx, start+n)
}

func (c *Client) LatestBlockHeight(ctx context.Context) (int64, error) {
	resp, err := c.Status(ctx)
	if err != nil {
		return 0, err
	}
	return resp.SyncInfo.LatestBlockHeight, nil
}

func (c *Client) Status(ctx context.Context) (*cmttypes.ResultStatus, error) {
	return c.cosmosCtx.Client.Status(ctx)
}

func (c *Client) WaitForBlockHeight(ctx context.Context, h int64) error {
	ticker := time.NewTicker(time.Millisecond * 250)
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
			return fmt.Errorf("timeout exceeded waiting for block, err: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) sendTx(ctx context.Context, queryMetaId uint64, msg ...sdk.Msg) (*cmttypes.ResultTx, error) {
	telemetry.IncrCounter(1, "daemon_sending_txs", "called")

	// Track success status for defer cleanup
	var txSuccess bool = false

	// Always reset commitedIds on any error, unless explicitly successful
	defer func() {
		if !txSuccess && queryMetaId != 0 {
			mutex.Lock()
			delete(commitedIds, queryMetaId)
			mutex.Unlock()
		}
	}()

	block, err := c.CmtService.GetLatestBlock(ctx, &cmtservice.GetLatestBlockRequest{})
	if err != nil {
		return nil, fmt.Errorf("error getting block: %w", err)
	}
	txf := newFactory(c.cosmosCtx)
	_, nonce, err := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, c.cosmosCtx.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting account number and sequence: %w", err)
	}

	// handle sequence conflicts for concurrent transaction submission
	mutex.Lock()
	if nonce <= lastSequenceUsed {
		// if chain sequence hasn't advanced, increment to avoid conflicts
		c.logger.Info(fmt.Sprintf("sequence conflict detected, sequence queried: %d, using incremented sequence: %d", nonce, lastSequenceUsed+1))
		nonce = lastSequenceUsed + 1
	}
	lastSequenceUsed = nonce
	mutex.Unlock()

	txf = txf.WithSequence(nonce).WithGasPrices(c.minGasFee).WithTimeoutHeight(uint64(block.SdkBlock.Header.Height + 2))
	txf, err = txf.Prepare(c.cosmosCtx)
	if err != nil {
		return nil, fmt.Errorf("error preparing transaction factory: %w", err)
	}

	txn, err := txf.BuildUnsignedTx(msg...)
	if err != nil {
		return nil, fmt.Errorf("error building unsigned transaction: %w", err)
	}
	if err = tx.Sign(c.cosmosCtx.CmdContext, txf, c.cosmosCtx.FromName, txn, true); err != nil {
		return nil, fmt.Errorf("error when signing transaction: %w", err)
	}

	txBytes, err := c.cosmosCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return nil, fmt.Errorf("error encoding transaction: %w", err)
	}
	res, err := c.cosmosCtx.BroadcastTx(txBytes)
	if err := handleBroadcastResult(res, err); err != nil {
		// check for sequence mismatch error and reset tracking if needed
		if res != nil && res.Code == 32 {
			c.handleSequenceError(nonce)
		}
		return nil, fmt.Errorf("error broadcasting transaction result: %w", err)
	}

	txnResponse, err := c.WaitForTx(ctx, res.TxHash)
	if err != nil {
		return nil, fmt.Errorf("error waiting for transaction: %w", err)
	}
	for _, event := range txnResponse.TxResult.Events {
		if event.Type == "new_report" {
			for _, attr := range event.Attributes {
				c.logger.Info("NewReport", attr.Key, attr.Value)
			}
		}
	}
	c.logger.Info(fmt.Sprintf("transaction hash: %s", res.TxHash))
	c.logger.Info(fmt.Sprintf("response after submit message: %d", txnResponse.TxResult.Code))
	if txnResponse.TxResult.Code == 0 {
		txSuccess = true // Prevent defer cleanup - keep queryMeta marked as committed
		telemetry.IncrCounter(1, "daemon_sending_txs", "success")
		telemetry.IncrCounterWithLabels([]string{"daemon_tx_gas_used_count"}, float32(txnResponse.TxResult.GasUsed), []metrics.Label{{Name: "chain_id", Value: c.cosmosCtx.ChainID}})
	}
	// If txSuccess stays false, defer will reset commitedIds[queryMetaId]

	return txnResponse, nil
}

func (c *Client) SetGasPrice(ctx context.Context) error {
	gfResponse, err := c.GlobalfeeClient.MinimumGasPrices(ctx, &globalfeetypes.QueryMinimumGasPricesRequest{})
	if err != nil {
		return fmt.Errorf("getting minimum gas price (globalfee): %w", err)
	}
	localPrice, err := sdk.ParseDecCoins(c.minGasFee)
	if err != nil {
		return fmt.Errorf("parsing local gas price: %w", err)
	}

	p := gasprice(gfResponse.MinimumGasPrices, localPrice)
	if p.IsZero() {
		return fmt.Errorf("unable to set gas price, global and local gas prices are zero")
	}
	c.minGasFee = p.String()
	return nil
}

func gasprice(local, global sdk.DecCoins) sdk.DecCoin {
	_local := sdk.NewDecCoin("loya", math.ZeroInt())
	for _, coin := range local {
		if coin.Denom == "loya" && coin.Amount.GT(math.LegacyZeroDec()) {
			_local = coin
		}
	}
	_global := sdk.NewDecCoin("loya", math.ZeroInt())
	for _, coin := range global {
		if coin.Denom == "loya" && coin.Amount.GT(math.LegacyZeroDec()) {
			_global = coin
		}
	}

	return sdk.DecCoin{
		Denom:  "loya",
		Amount: math.LegacyMaxDec(_local.Amount, _global.Amount),
	}
}

// func getcommitId(events []abcitypes.Event) (uint64, error) {
// 	for _, event := range events {
// 		if event.Type == "new_commit" {
// 			for _, attr := range event.Attributes {
// 				if attr.Key == "commit_id" {
// 					value, err := strconv.Atoi(attr.Value)
// 					if err != nil {
// 						return 0, err
// 					}
// 					return uint64(value), nil
// 				}
// 			}
// 		}
// 	}
// 	return 0, fmt.Errorf("commit_id not found")
// }

// handleSequenceError resets global sequence tracking by querying current chain state
func (c *Client) handleSequenceError(failedSequence uint64) {
	// query current sequence directly from chain instead of parsing error strings
	_, currentChainSeq, err := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, c.cosmosCtx.FromAddress)
	if err != nil {
		c.logger.Error(fmt.Sprintf("failed to query chain sequence during error recovery: %v", err))
		return
	}

	mutex.Lock()
	// only reset if our tracking is ahead of chain state (indicating failed transactions)
	if lastSequenceUsed >= currentChainSeq {
		lastSequenceUsed = currentChainSeq - 1
		c.logger.Warn(fmt.Sprintf("sequence error recovery - failed seq: %d, chain seq: %d, reset tracking to: %d",
			failedSequence, currentChainSeq, currentChainSeq-1))
	} else {
		c.logger.Info(fmt.Sprintf("sequence tracking already correct - failed seq: %d, chain seq: %d, tracking: %d",
			failedSequence, currentChainSeq, lastSequenceUsed))
	}
	mutex.Unlock()
}
