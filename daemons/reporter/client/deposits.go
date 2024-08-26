package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (c *Client) deposits() (queryData []byte, value string, err error) {
	oldestDeposit, err := c.TokenDepositsCache.GetOldestReport()
	if err != nil {
		return nil, "", fmt.Errorf("no pending deposits")
	}

	return oldestDeposit.QueryData, hex.EncodeToString(oldestDeposit.Value), nil
}

type SubmissionMgr struct {
	commits      map[string]*time.Timer
	mu           sync.Mutex
	daemonClient *Client
}

func NewSubmissionManager(client *Client) *SubmissionMgr {
	return &SubmissionMgr{
		commits:      make(map[string]*time.Timer),
		daemonClient: client,
	}
}

func (m *SubmissionMgr) AddDepositCommit(ctx context.Context, queryId string, commit Commit, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Schedule the submit tx action after the delay
	timer := time.AfterFunc(delay, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		msg := oracletypes.MsgSubmitValue{
			Creator:   m.daemonClient.accAddr.String(),
			QueryData: commit.querydata,
			Value:     commit.value,
			Salt:      commit.salt,
			CommitId:  commit.Id,
		}
		resp, err := m.daemonClient.sendTx(ctx, &msg)
		if err != nil {
			m.daemonClient.logger.Error("sending submit deposit transaction", "error", err)
		}
		fmt.Println("response code after submit deposit transaction message", resp.TxResult.Code)
		fmt.Println("response tx hash after submit deposit transaction message", resp.Hash.String())
		// Remove the item from the map
		delete(m.commits, queryId)
		delete(depositMeta, queryId)
	})

	// Store the timer
	m.commits[queryId] = timer
	depositMeta[queryId] = commit
}
