package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/tellor-io/layer/lib/metrics"
	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
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

func (c *Client) GenerateDepositMessages(ctx context.Context) error {
	depositQuerydata, value, err := c.deposits()
	if err != nil {
		if err.Error() == "no pending deposits" {
			return nil
		}
		return fmt.Errorf("error getting deposits: %w", err)
	}

	queryId := hex.EncodeToString(utils.QueryIDFromData(depositQuerydata))
	mutex.RLock()
	depositReported := depositReportMap[queryId]
	mutex.RUnlock()

	if depositReported {
		c.logger.Info("Skipping already reported deposit", "queryId", queryId)
		return nil
	}

	msg := &oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: depositQuerydata,
		Value:     value,
	}

	telemetry.IncrCounterWithLabels([]string{"daemon_bridge_deposit", "found"}, 1, []metrics.Label{{Name: "chain_id", Value: c.cosmosCtx.ChainID}})
	c.txChan <- TxChannelInfo{Msg: msg, isBridge: true, NumRetries: 5}

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
	value, err := c.median(qd)
	if err != nil {
		return fmt.Errorf("error getting median from median client': %w", err)
	}

	msg := &oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: qd,
		Value:     value,
	}

	c.txChan <- TxChannelInfo{Msg: msg, isBridge: false, NumRetries: 0}

	mutex.Lock()
	commitedIds[querymeta.Id] = true
	mutex.Unlock()

	c.LogProcessStats()

	return nil
}

func (c *Client) HandleBridgeDepositTxInChannel(ctx context.Context, data TxChannelInfo) {
	resp, err := c.sendTx(ctx, data.Msg)
	if err != nil {
		c.logger.Error("submitting deposit report transaction",
			"error", err,
			"attemptsLeft", data.NumRetries)

		if data.NumRetries == 0 {
			// Don't mark as reported if all retries failed
			c.logger.Error(fmt.Sprintf("failed to submit deposit after all allotted attempts attempts: %v", err))
			return
		}

		data.NumRetries--
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
			"queryId", queryId)
		return
	}

	// Remove oldest deposit report from cache
	c.TokenDepositsCache.RemoveOldestReport()

	// Only mark as reported if transaction was successful
	mutex.Lock()
	depositReportMap[hex.EncodeToString(queryId)] = true
	mutex.Unlock()

	telemetry.IncrCounterWithLabels([]string{"daemon_bridge_deposit", "reported"}, 1, []metrics.Label{{Name: "chain_id", Value: c.cosmosCtx.ChainID}})
	c.logger.Info(fmt.Sprintf("Response from bridge tx report: %v", resp.TxResult))
}

func (c *Client) BroadcastTxMsgToChain() {
	for obj := range c.txChan {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		done := make(chan struct{})
		go func() {
			defer close(done)
			if !obj.isBridge {
				_, err := c.sendTx(ctx, obj.Msg)
				if err != nil {
					c.logger.Error(fmt.Sprintf("Error sending tx: %v", err))
				}
			} else {
				c.HandleBridgeDepositTxInChannel(ctx, obj)
			}
		}()

		select {
		case <-done:
			cancel()
		case <-ctx.Done():
			c.logger.Error("broadcasting tx timed out")
			cancel()
		}
	}
}
