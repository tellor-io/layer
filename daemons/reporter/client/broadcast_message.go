package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
)

// cycle list
const (
	ethQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	btcQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	trbQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
)

var (
	eth, _ = utils.QueryBytesFromString(ethQueryData)
	btc, _ = utils.QueryBytesFromString(btcQueryData)
	trb, _ = utils.QueryBytesFromString(trbQueryData)
)

func (c *Client) generateDepositmessages(ctx context.Context, bg *sync.WaitGroup) error {
	defer bg.Done()
	depositQuerydata, value, err := c.deposits()
	if err != nil {
		return fmt.Errorf("error getting deposits: %w", err)
	}
	queryId := hex.EncodeToString(utils.QueryIDFromData(depositQuerydata))
	if !depositCommitMap[queryId] {
		salt, err := oracleutils.Salt(32)
		if err != nil {
			return fmt.Errorf("error generating salt: %w", err)
		}
		hash := oracleutils.CalculateCommitment(value, salt)
		msgCommit := &oracletypes.MsgCommitReport{
			Creator:   c.accAddr.String(),
			QueryData: depositQuerydata,
			Hash:      hash,
		}
		resp, err := c.sendTx(ctx, msgCommit)
		if err != nil {
			return fmt.Errorf("error sending tx: %w", err)
		}
		fmt.Println("response after deposit commit message", resp.TxResult.Code)
		fmt.Println("deposit commit transaction hash", resp.Hash.String())
		commitId, err := getcommitId(resp.TxResult.Events)
		if err != nil {
			return fmt.Errorf("error getting commit id from response: %w", err)
		}
		fmt.Println("commit id", commitId)
		depositCommitMap[queryId] = true
		commit := Commit{
			querydata: depositQuerydata,
			value:     value,
			salt:      salt,
			Id:        commitId,
		}
		queryresp, err := c.OracleQueryClient.GetQuery(ctx, &oracletypes.QueryGetQueryRequest{QueryId: queryId, Id: commitId})
		if err != nil {
			return fmt.Errorf("error getting query meta: %w", err)
		}
		block, err := c.cosmosCtx.Client.Block(ctx, nil)
		if err != nil {
			return fmt.Errorf("error getting block: %w", err)
		}
		expiry := queryresp.Query.Expiration.Sub(block.Block.Time)
		// add error handling
		c.SubMgr.AddDepositCommit(ctx, queryId, commit, expiry)
	}
	return nil
}

func (c *Client) generateExternalMessages(ctx context.Context, filepath string, bg *sync.WaitGroup) error {
	defer bg.Done()
	jsonFile, err := os.ReadFile(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("error reading from file: %w", err)
	}
	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("error deleting transactions file: %w", err)
	}
	tx, err := c.cosmosCtx.TxConfig.TxJSONDecoder()(jsonFile)
	if err != nil {
		return fmt.Errorf("error decoding json file: %w", err)
	}
	msgs := tx.GetMsgs()

	resp, err := c.sendTx(ctx, msgs...)
	if err != nil {
		return fmt.Errorf("error sending tx: %w", err)
	}
	fmt.Println("response after external message", resp.TxResult.Code)

	return nil
}

func (c *Client) CyclelistMessages(ctx context.Context, qd []byte, bg *sync.WaitGroup) error {
	defer bg.Done()
	querydata, querymeta, err := c.CurrentQuery(ctx)
	if err != nil {
		// log error
		c.logger.Error("getting current query", "error", err)
	}
	for !bytes.Equal(querydata, qd) || commitedIds[querymeta.Id] {
		time.Sleep(time.Millisecond * 500)
		querydata, querymeta, err = c.CurrentQuery(ctx)
		if err != nil {
			// log error
			c.logger.Error("getting current query on recursion", "error", err)
		}
	}
	value, err := c.median(querydata)
	if err != nil {
		return fmt.Errorf("error getting median from median client': %w", err)
	}
	salt, err := oracleutils.Salt(32)
	if err != nil {
		return fmt.Errorf("error generating salt: %w", err)
	}

	hash := oracleutils.CalculateCommitment(value, salt)
	commitmsg := &oracletypes.MsgCommitReport{
		Creator:   c.accAddr.String(),
		QueryData: querydata,
		Hash:      hash,
	}

	resp, err := c.sendTx(ctx, commitmsg)
	if err != nil {
		return fmt.Errorf("error sending tx: %w", err)
	}
	if resp.TxResult.Code != 0 {
		return fmt.Errorf("commit transaction failed with code %d", resp.TxResult.Code)
	}
	fmt.Println("response after commit message", resp.TxResult.Code)
	time.Sleep(querymeta.RegistrySpecTimeframe)
	msg := &oracletypes.MsgSubmitValue{
		Creator:   c.accAddr.String(),
		QueryData: querydata,
		Value:     value,
		Salt:      salt,
		CommitId:  querymeta.Id,
	}

	resp, err = c.sendTx(ctx, msg)
	if err != nil {
		return fmt.Errorf("error sending tx: %w", err)
	}
	fmt.Println("response after submit message", resp.TxResult.Code)
	commitedIds[querymeta.Id] = true

	return nil
}
