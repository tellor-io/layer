package client

import (
	"context"
	"encoding/hex"
	"fmt"

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

func (c *Client) generateDepositmessages(ctx context.Context) error {
	depositQuerydata, value, err := c.deposits()
	if err != nil {
		if err.Error() == "no pending deposits" {
			return nil
		}
		return fmt.Errorf("error getting deposits: %w", err)
	}

	queryId := hex.EncodeToString(utils.QueryIDFromData(depositQuerydata))
	if depositReportMap[queryId] {
		return fmt.Errorf("already reported for this bridge deposit tx")
	}
	msg := oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: depositQuerydata,
		Value:     value,
	}
	resp, err := c.sendTx(ctx, &msg)
	if err != nil {
		c.logger.Error("sending submit deposit transaction", "error", err)
	}
	c.logger.Info(fmt.Sprintf("Response from bridge tx report: %v", resp.TxResult))
	depositReportMap[queryId] = true

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

func (c *Client) GenerateAndBroadcastSpotPriceReports(ctx context.Context, qds [][]byte, querymetaIds []uint64) error {
	reportMsgs := make([]*oracletypes.MsgSubmitValue, 0)
	for i := 0; i < len(qds); i++ {
		value, err := c.median(qds[i])
		if err != nil {
			c.logger.Info("error getting median from median client: ", err)
			continue
		}

		msg := &oracletypes.MsgSubmitValue{
			Creator:   c.accAddr.String(),
			QueryData: qds[i],
			Value:     value,
		}
		reportMsgs = append(reportMsgs, msg)
		commitedIds[querymetaIds[i]] = true
	}

	if len(reportMsgs) == 0 {
		return fmt.Errorf("error getting all medians")
	}

	resp, err := c.sendTx(ctx, reportMsgs)
	if err != nil {
		return fmt.Errorf("error sending tx: %w", err)
	}
	fmt.Println("response after submit message", resp.TxResult.Code)

	return nil
}

func (c *Client) GenerateAndBroadcastSpotPriceReport(ctx context.Context, qd []byte, querymetaId uint64) error {
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
	commitedIds[querymetaId] = true

	return nil
}

func (c *Client) GenerateAndBroadcastTippedQueries(ctx context.Context, qds [][]byte, querymetaIds []uint64) error {
	if len(qds) != len(querymetaIds) {
		c.logger.Error("ERROR length of query data array and querymeta Id array must be equal")
		return fmt.Errorf("length of array parameters must be equal")
	}

	submitValueMsgs := make([]*oracletypes.MsgSubmitValue, 0)
	for i := 0; i < len(qds); i++ {
		value, err := c.median(qds[i])
		if err != nil {
			return fmt.Errorf("error getting median from median client': %w", err)
		}

		msg := &oracletypes.MsgSubmitValue{
			Creator:   c.accAddr.String(),
			QueryData: qds[i],
			Value:     value,
		}
		submitValueMsgs = append(submitValueMsgs, msg)
	}

	resp, err := c.sendTx(ctx, submitValueMsgs)
	if err != nil {
		return fmt.Errorf("error sending tx: %w", err)
	}
	fmt.Println("response after submit message", resp.TxResult.Code)
	for i := 0; i < len(querymetaIds); i++ {
		commitedIds[querymetaIds[i]] = true
	}

	return nil
}

// func (c *Client) GenerateAndBroadcastSpotPriceReport(ctx context.Context, qd []byte, querymeta *oracletypes.QueryMeta) error {
// 	value, err := c.median(qd)
// 	if err != nil {
// 		return fmt.Errorf("error getting median from median client': %w", err)
// 	}

// 	msg := &oracletypes.MsgSubmitValue{
// 		Creator:   c.accAddr.String(),
// 		QueryData: qd,
// 		Value:     value,
// 	}

// 	resp, err := c.sendTx(ctx, msg)
// 	if err != nil {
// 		return fmt.Errorf("error sending tx: %w", err)
// 	}
// 	fmt.Println("response after submit message", resp.TxResult.Code)
// 	commitedIds[querymeta.Id] = true

// 	return nil
// }
