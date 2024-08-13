package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	cmttypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
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
	bz, err := hex.DecodeString(hash)
	if err != nil {
		return nil, fmt.Errorf("unable to decode tx hash '%s'; err: %w", hash, err)
	}
	for {
		resp, err := c.cosmosCtx.Client.Tx(ctx, bz, false)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				// Tx not found, wait for next block and try again
				err := c.WaitForNextBlock(ctx)
				if err != nil {
					return nil, fmt.Errorf("waiting for next block: err: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf("fetching tx '%s'; err: %w", hash, err)
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
	ticker := time.NewTicker(time.Millisecond * 500)
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
			fmt.Println("Waiting for block height", h, "current height", latestHeight)
		}
	}
}

var mus sync.Mutex

func (c *Client) sendTx(ctx context.Context, msg []sdk.Msg, nonce uint64) error {
	mus.Lock()
	defer mus.Unlock()
	block, err := c.cosmosCtx.Client.Block(ctx, nil)
	if err != nil {
		fmt.Println("Error getting block: ", err)
	}
	txf := newFactory(c.cosmosCtx)
	txf = txf.WithSequence(nonce)
	txf = txf.WithTimeoutHeight(uint64(block.Block.Header.Height + 2))
	c.logger.Info("Transaction nonce", "nonce", nonce)

	txf, err = txf.Prepare(c.cosmosCtx)
	if err != nil {
		return fmt.Errorf("error preparing transaction factory: %w", err)
	}

	txn, err := txf.BuildUnsignedTx(msg...)
	if err != nil {
		return fmt.Errorf("error building unsigned transaction: %w", err)
	}
	if err = tx.Sign(c.cosmosCtx.CmdContext, txf, c.cosmosCtx.FromName, txn, true); err != nil {
		return fmt.Errorf("error when signing transaction: %w", err)
	}

	txBytes, err := c.cosmosCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return fmt.Errorf("error encoding transaction: %w", err)
	}
	res, err := c.cosmosCtx.BroadcastTx(txBytes)
	if err := handleBroadcastResult(res, err); err != nil {
		return fmt.Errorf("error broadcasting transaction result: %w", err)
	}
	txnReponse, err := c.WaitForTx(ctx, res.TxHash)
	if err != nil {
		return fmt.Errorf("error waiting for transaction: %w", err)
	}
	c.logger.Info("TxResult", "result", txnReponse.TxResult)

	return nil
}
