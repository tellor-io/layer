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

// cycle list
// const (
// 	ethQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// 	btcQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// 	trbQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// )

// var (
// 	eth, _ = utils.QueryBytesFromString(ethQueryData)
// 	btc, _ = utils.QueryBytesFromString(btcQueryData)
// 	trb, _ = utils.QueryBytesFromString(trbQueryData)
// )

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

// func (c *Client) generateExternalMessages(ctx context.Context, filepath string, bg *sync.WaitGroup) error {
// 	defer bg.Done()
// 	jsonFile, err := os.ReadFile(filepath)
// 	if err != nil {
// 		if errors.Is(err, os.ErrNotExist) {
// 			return nil
// 		}
// 		return fmt.Errorf("error reading from file: %w", err)
// 	}
// 	if err := os.Remove(filepath); err != nil {
// 		return fmt.Errorf("error deleting transactions file: %w", err)
// 	}
// 	tx, err := c.cosmosCtx.TxConfig.TxJSONDecoder()(jsonFile)
// 	if err != nil {
// 		return fmt.Errorf("error decoding json file: %w", err)
// 	}
// 	msgs := tx.GetMsgs()

// 	resp, err := c.sendTx(ctx, msgs...)
// 	if err != nil {
// 		return fmt.Errorf("error sending tx: %w", err)
// 	}
// 	fmt.Println("response after external message", resp.TxResult.Code)

// 	return nil
// }

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

func (c *Client) BroadcastTxMsgToChain() {
	for obj := range c.txChan {
		// submit transaction in goroutine without waiting for completion
		go func(txInfo TxChannelInfo) {
			ctx, cancel := context.WithTimeout(context.Background(), 4500*time.Millisecond)
			defer cancel()

			if !txInfo.isBridge {
				_, err := c.sendTx(ctx, txInfo.QueryMetaId, txInfo.Msg)
				if err != nil {
					c.logger.Error(fmt.Sprintf("Error sending tx: %v", err))
				}
			} else {
				c.HandleBridgeDepositTxInChannel(ctx, txInfo)
			}
		}(obj)

		// log channel status and immediately continue to next transaction
		c.logger.Info(fmt.Sprintf("Tx in Channel: %d", len(c.txChan)))
	}
}
