package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tellor-io/layer/daemons/lib/metrics"
	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

const (
	bridgeDepositMaxRetries = 10 // Bridge deposits have ~1 hour window, so more retries are acceptable
)

func (c *Client) GenerateDepositMessages(ctx context.Context) error {
	depositQuerydata, value, err := c.deposits()
	if err != nil {
		if err.Error() == "no pending deposits" {
			return nil
		}
		return fmt.Errorf("error getting deposits: %w", err)
	}
	msg := &oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: depositQuerydata,
		Value:     value,
	}

	telemetry.IncrCounterWithLabels([]string{"daemon_bridge_deposit", "found"}, 1, []metrics.Label{{Name: "chain_id", Value: c.cosmosCtx.ChainID}})
	c.txChan <- TxChannelInfo{Msg: msg, isBridge: true, NumRetries: bridgeDepositMaxRetries, QueryMetaId: 0}

	return nil
}

func (c *Client) GenerateAndBroadcastSpotPriceReport(ctx context.Context, qd []byte, querymeta *oracletypes.QueryMeta) error {
	encodedValue, rawPrice, err := c.median(qd)
	if err != nil {
		return fmt.Errorf("error getting median from median client': %w", err)
	}

	// Check price guard before submitting
	// rawPrice is 0 for custom queries
	if c.PriceGuard.enabled && rawPrice > 0 {
		shouldSubmit, reason := c.PriceGuard.ShouldSubmit(qd, rawPrice)
		// Update baseline if submission allowed OR if configured to update on block
		if shouldSubmit || c.PriceGuard.UpdateOnBlocked() {
			c.PriceGuard.UpdateLastPrice(qd, rawPrice)
		}

		if !shouldSubmit {
			if c.PriceGuard.UpdateOnBlocked() {
				// only update if price guard is configured to update on blocked
				// help prevent tipped queries from getting stuck if no reports get made
				mutex.Lock()
				commitedIds[querymeta.Id] = true
				mutex.Unlock()
			}

			querydatastr := hex.EncodeToString(qd)
			queryIdHex := utils.QueryIDFromData(qd)

			pair := ""
			for _, marketParam := range c.MarketParams {
				if marketParam.QueryData == querydatastr {
					pair = marketParam.Pair
					break
				}
			}

			if pair != "" {
				return fmt.Errorf("price guard blocked submission for %s: %s", pair, reason)
			}
			return fmt.Errorf("price guard blocked submission for queryId %x: %s", queryIdHex, reason)
		}
	}

	msg := &oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: qd,
		Value:     encodedValue,
	}

	c.txChan <- TxChannelInfo{
		Msg:         msg,
		isBridge:    false,
		NumRetries:  0,
		QueryMetaId: querymeta.Id,
	}

	// Mark as committed immediately to prevent duplicate processing
	mutex.Lock()
	commitedIds[querymeta.Id] = true
	mutex.Unlock()

	c.LogProcessStats()

	return nil
}

func (c *Client) HandleBridgeDepositTxInChannel(ctx context.Context, data TxChannelInfo) {
	resp, err := c.sendTx(ctx, 0, data.Msg) // 0 = no queryMeta tracking for bridge transactions
	if err != nil {
		c.logger.Error("submitting deposit report transaction",
			"error", err,
			"attemptsLeft", data.NumRetries)

		if data.NumRetries == 0 {
			// Don't mark as reported if all retries failed
			c.logger.Error(fmt.Sprintf("failed to submit deposit after all allotted attempts attempts: %v", err))
			// Remove oldest deposit report from cache
			c.TokenDepositsCache.RemoveOldestReport()
			return
		}

		data.NumRetries--

		// For unordered transactions, we don't need to handle concurrent transaction limits

		c.txChan <- data

		return
	}

	var bridgeDepositMsg *oracletypes.MsgSubmitValue
	var queryId []byte
	if msg, ok := data.Msg.(*oracletypes.MsgSubmitValue); ok {
		bridgeDepositMsg = msg
	} else {
		c.logger.Error("Could not go from sdk.Msg to types.MsgSubmitValue")
		return
	}

	queryId = utils.QueryIDFromData(bridgeDepositMsg.GetQueryData())

	// Check transaction success
	if resp.TxResult.Code != 0 {
		c.logger.Error("deposit report transaction failed",
			"code", resp.TxResult.Code,
			"queryId", queryId,
			"log", resp.TxResult.Log)
	}

	// Remove oldest deposit report from cache
	c.TokenDepositsCache.RemoveOldestReport()

	telemetry.IncrCounterWithLabels([]string{"daemon_bridge_deposit", "reported"}, 1, []metrics.Label{{Name: "chain_id", Value: c.cosmosCtx.ChainID}})
	c.logger.Info(fmt.Sprintf("Response from bridge tx report: %v", resp.TxResult))
}

func (c *Client) BroadcastTxMsgToChain(ctx context.Context) {
	defer c.broadcastWg.Wait()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("BroadcastTxMsgToChain: context cancelled, exiting")
			return
		case obj, ok := <-c.txChan:
			if !ok {
				c.logger.Info("BroadcastTxMsgToChain: channel closed, exiting")
				return
			}
			// submit transaction in goroutine with proper tracking
			c.broadcastWg.Add(1)
			go func(txInfo TxChannelInfo) {
				defer c.broadcastWg.Done()
				txCtx, cancel := context.WithTimeout(ctx, 4500*time.Millisecond)
				defer cancel()

				if !txInfo.isBridge {
					_, err := c.sendTx(txCtx, txInfo.QueryMetaId, txInfo.Msg)
					if err != nil {
						c.logger.Error(fmt.Sprintf("Error sending tx: %v", err))
					}
				} else {
					c.HandleBridgeDepositTxInChannel(txCtx, txInfo)
				}
			}(obj)

			// log channel status and immediately continue to next transaction
			c.logger.Info(fmt.Sprintf("Tx in Channel: %d", len(c.txChan)))
		}
	}
}
