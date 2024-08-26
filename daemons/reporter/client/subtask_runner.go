package client

import (
	"context"
	"sync"
)

type SubTaskRunner interface {
	RunReporterDaemonTaskLoop(
		ctx context.Context,
		client *Client,
	) error
}

type SubTaskRunnerImpl struct{}

// Ensure SubTaskRunnerImpl implements the SubTaskRunner interface.
var _ SubTaskRunner = (*SubTaskRunnerImpl)(nil)

func (s *SubTaskRunnerImpl) RunReporterDaemonTaskLoop(
	ctx context.Context,
	daemonClient *Client,
) error {
	var bg sync.WaitGroup

	bg.Add(3)

	go func() {
		err := daemonClient.EthMessages(ctx, &bg)
		if err != nil {
			daemonClient.logger.Error("Generating eth messages", "error", err)
		}
	}()
	go func() {
		err := daemonClient.BTCMessages(ctx, &bg)
		if err != nil {
			daemonClient.logger.Error("Generating btc messages", "error", err)
		}
	}()
	go func() {
		err := daemonClient.TRBMessages(ctx, &bg)
		if err != nil {
			daemonClient.logger.Error("Generating trb messages", "error", err)
		}
	}()

	bg.Add(1)
	go func() {
		err := daemonClient.generateDepositmessages(ctx, &bg)
		if err != nil {
			daemonClient.logger.Error("Generating deposit messages", "error", err)
		}
	}()

	bg.Add(1)
	go func() {
		err := daemonClient.generateExternalMessages(ctx, "unsignedtx.json", &bg)
		if err != nil {
			daemonClient.logger.Error("Generating external messages", "error", err)
		}
	}()

	bg.Wait()

	return nil
}
