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

	// Check if the reporter is created
	go func() {
		for !reporterCreated {
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

	go func() {
		err := daemonClient.generateCommitMessages(ctx, commitCh)
		if err != nil {
			daemonClient.logger.Error("Generating commit messages", "error", err)
		}
	}()
	go func() {
		err := daemonClient.generateSubmitMessages(ctx, submitCh)
		if err != nil {
			daemonClient.logger.Error("Generating submit messages", "error", err)
		}
	}()
	go collectMessages(commitCh, submitCh, broadcastTrigger)
	go func() {
		err := daemonClient.generateDepositCommits(commitCh)
		if err != nil {
			daemonClient.logger.Error("Generating deposit commits", "error", err)
		}
	}()
	go func() {
		err := daemonClient.generateDepositSubmits(ctx, submitCh)
		if err != nil {
			daemonClient.logger.Error("Generating deposit submits", "error", err)
		}
	}()
	go func() {
		err := daemonClient.generateExternalMessages("unsignedtx.json", broadcastTrigger)
		if err != nil {
			daemonClient.logger.Error("Generating external messages", "error", err)
		}
	}()
	go func() {
		err := daemonClient.broadcastMessages(ctx, broadcastTrigger)
		if err != nil {
			daemonClient.logger.Error("Broadcasting messages", "error", err)
		}
	}()

	return nil
}
