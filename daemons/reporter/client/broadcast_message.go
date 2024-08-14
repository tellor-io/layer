package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (c *Client) generateDepositCommits(commitCh chan<- sdk.Msg) error {
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
		fmt.Println("Salt:", salt)
		fmt.Println("Value:", value)
		fmt.Println("Hash: ")
		hash := oracleutils.CalculateCommitment(value, salt)
		msgCommit := &oracletypes.MsgCommitReport{
			Creator:   c.accAddr.String(),
			QueryData: depositQuerydata,
			Hash:      hash,
		}
		commitCh <- msgCommit

		depositCommitMap[queryId] = true
		depositMeta[queryId] = commit{
			querydata:  depositQuerydata,
			value:      value,
			salt:       salt,
			expiration: time.Now().Add(time.Hour), // todo: have to change this to make it accurate
		}
	}
	return nil
}

func (c *Client) generateDepositSubmits(ctx context.Context, submitCh chan<- sdk.Msg) error {
	for id, commit := range depositMeta {
		block, err := c.cosmosCtx.Client.Block(ctx, nil)
		if err != nil {
			return fmt.Errorf("error getting block: %w", err)
		}
		blockTime := block.Block.Header.Time
		commitWindowExpired := commit.expiration.Before(blockTime)
		inrevealWindow := commit.expiration.Add(time.Second * 3).After(blockTime)

		if commitWindowExpired && inrevealWindow {
			msg := &oracletypes.MsgSubmitValue{
				Creator:   c.accAddr.String(),
				QueryData: commit.querydata,
				Value:     commit.value,
				Salt:      commit.salt,
			}

			submitCh <- msg
			delete(depositMeta, id)
		}
	}

	return nil
}

func (c *Client) generateCommitMessages(ctx context.Context, commitCh chan<- sdk.Msg) error {
	querydata, querymeta, err := c.CurrentQuery(ctx)
	if err != nil {
		return fmt.Errorf("error calling 'CurrentQuery': %w", err)
	}

	if !commitedIds[querymeta.Id] {
		value, err := c.median(querydata)
		if err != nil {
			return fmt.Errorf("error getting median from median client': %w", err)
		}
		salt, err := oracleutils.Salt(32)
		if err != nil {
			return fmt.Errorf("error generating salt: %w", err)
		}
		fmt.Println("Salt:", salt)
		fmt.Println("Value:", value)
		fmt.Println("Hash: ")
		hash := oracleutils.CalculateCommitment(value, salt)
		fmt.Println("Hash: ", hash)
		commitmsg := &oracletypes.MsgCommitReport{
			Creator:   c.accAddr.String(),
			QueryData: querydata,
			Hash:      hash,
		}

		commitCh <- commitmsg

		commitedIds[querymeta.Id] = true
		idToCommit[int64(querymeta.Id)] = commit{
			querydata:  querydata,
			value:      value,
			salt:       salt,
			expiration: querymeta.Expiration,
		}

	}

	return nil
}

func (c *Client) generateSubmitMessages(ctx context.Context, submitCh chan<- sdk.Msg) error {
	for id, commit := range idToCommit {
		block, err := c.cosmosCtx.Client.Block(ctx, nil)
		if err != nil {
			return fmt.Errorf("error getting block: %w", err)
		}
		blockTime := block.Block.Header.Time
		commitWindowExpired := commit.expiration.Before(blockTime)
		inrevealWindow := commit.expiration.Add(time.Second * 3).After(blockTime)

		if commitWindowExpired && inrevealWindow {
			msg := &oracletypes.MsgSubmitValue{
				Creator:   c.accAddr.String(),
				QueryData: commit.querydata,
				Value:     commit.value,
				Salt:      commit.salt,
			}

			submitCh <- msg
			delete(idToCommit, id)
		}
	}

	return nil
}

func collectMessages(chA, submitCh <-chan sdk.Msg, broadcastTrigger chan<- struct{}) {
	for {
		select {
		case msg := <-chA:
			bmu.Lock()
			messagesA = append(messagesA, msg)
			bmu.Unlock()

			broadcastTrigger <- struct{}{} // Trigger broadcast

		case msg := <-submitCh:
			bmu.Lock()
			messagesB = append(messagesB, msg)
			broadcastTrigger <- struct{}{}
			bmu.Unlock()
		}
	}
}

func (c *Client) broadcastMessages(ctx context.Context, broadcastTrigger <-chan struct{}) error {
	for range broadcastTrigger {
		bmu.Lock()
		if len(messagesA) > 0 || len(messagesB) > 0 || len(messagesC) > 0 {
			combinedMessages := append(messagesA, messagesB...)
			combinedMessages = append(combinedMessages, messagesC...)
			fmt.Println("Combined messages:", combinedMessages)
			_, seq, _ := c.cosmosCtx.AccountRetriever.GetAccountNumberSequence(c.cosmosCtx, c.accAddr)
			err := c.sendTx(ctx, combinedMessages, seq)
			if err != nil {
				return fmt.Errorf("error sending tx: %w", err)
			}

			messagesA = []sdk.Msg{}
			messagesB = []sdk.Msg{}
			messagesC = []sdk.Msg{}
		}
		bmu.Unlock()
	}
	return nil
}

func (c *Client) generateExternalMessages(filepath string, trigger chan<- struct{}) error {
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

	bmu.Lock()
	messagesC = msgs
	fmt.Println("External message:", msgs)
	bmu.Unlock()
	if len(messagesC) > 0 {
		trigger <- struct{}{}
	}

	return nil
}
