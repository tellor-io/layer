package client

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type SubTaskRunner interface {
	RunReporterDaemonTaskLoop(
		ctx context.Context,
		client *Client,
		commitCh chan sdk.Msg,
		submitCh chan sdk.Msg,
		broadcasttrigger chan struct{},
	) error
}

type SubTaskRunnerImpl struct{}

// Ensure SubTaskRunnerImpl implements the SubTaskRunner interface.
var _ SubTaskRunner = (*SubTaskRunnerImpl)(nil)

func (s *SubTaskRunnerImpl) RunReporterDaemonTaskLoop(
	ctx context.Context,
	daemonClient *Client,
	commitCh chan sdk.Msg,
	submitCh chan sdk.Msg,
	broadcastTrigger chan struct{},
) error {
	reporterCreated := false
	conditionCh := make(chan struct{})

	// Start a goroutine to monitor the condition
	go func() {
		for !reporterCreated {
			// Replace this with your actual condition check
			reporterCreated = daemonClient.checkReporter(ctx)
			if reporterCreated {
				close(conditionCh)
				err := daemonClient.WaitForNextBlock(ctx)
				if err != nil {
					daemonClient.logger.Error("Waiting for next block after creating a reporter", "error", err)
				}
			} else {
				time.Sleep(time.Second)
			}
		}
	}()

	// Wait for the condition to be met before starting the tasks
	<-conditionCh

	go daemonClient.generateCommitMessages(ctx, commitCh)
	go daemonClient.generateSubmitMessages(ctx, submitCh)
	go collectMessages(commitCh, submitCh, broadcastTrigger)
	go daemonClient.generateDepositCommits(commitCh)
	go daemonClient.generateDepositSubmits(ctx, submitCh)
	go daemonClient.broadcastMessages(ctx, broadcastTrigger)

	return nil
}
