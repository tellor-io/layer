package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

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

	msg := oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: depositQuerydata,
		Value:     value,
	}

	// Add retry logic for transaction sending
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := c.sendTx(ctx, &msg)
		if err != nil {
			c.logger.Error("submitting deposit report transaction",
				"error", err,
				"attempt", attempt,
				"queryId", queryId)

			if attempt == maxRetries {
				// Don't mark as reported if all retries failed
				return fmt.Errorf("failed to submit deposit after %d attempts: %w", maxRetries, err)
			}

			// Wait before retry with exponential backoff
			time.Sleep(time.Second * time.Duration(2^attempt))
			continue
		}

		// Check transaction success
		if resp.TxResult.Code != 0 {
			c.logger.Error("deposit report transaction failed",
				"code", resp.TxResult.Code,
				"queryId", queryId)
			// remove oldest deposit report and move on to next one
			c.TokenDepositsCache.RemoveOldestReport()
			return fmt.Errorf("transaction failed with code %d", resp.TxResult.Code)
		}

		// Remove oldest deposit report from cache
		c.TokenDepositsCache.RemoveOldestReport()

		// Only mark as reported if transaction was successful
		mutex.Lock()
		depositReportMap[queryId] = true
		mutex.Unlock()

		c.logger.Info(fmt.Sprintf("Response from bridge tx report: %v", resp.TxResult))

		return nil
	}

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

	resp, err := c.sendTx(ctx, msg)
	if err != nil {
		return fmt.Errorf("error sending tx: %w", err)
	}
	fmt.Println("response after submit message", resp.TxResult.Code)
	mutex.Lock()
	commitedIds[querymeta.Id] = true
	mutex.Unlock()

	c.LogProcessStats()

	return nil
}
