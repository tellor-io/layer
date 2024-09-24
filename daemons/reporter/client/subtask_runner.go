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

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		err := daemonClient.CyclelistMessages(ctx, eth)
		if err != nil {
			daemonClient.logger.Error("Generating eth messages", "error", err)
		}
	}(&bg)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		err := daemonClient.CyclelistMessages(ctx, btc)
		if err != nil {
			daemonClient.logger.Error("Generating btc messages", "error", err)
		}
	}(&bg)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		err := daemonClient.CyclelistMessages(ctx, trb)
		if err != nil {
			daemonClient.logger.Error("Generating trb messages", "error", err)
		}
	}(&bg)

	bg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		err := daemonClient.generateDepositmessages(ctx)
		if err != nil {
			daemonClient.logger.Error("Generating deposit messages", "error", err)
		}
	}(&bg)

	bg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		err := daemonClient.generateExternalMessages(ctx, "unsignedtx.json")
		if err != nil {
			daemonClient.logger.Error("Generating external messages", "error", err)
		}
	}(&bg)

	bg.Wait()

	return nil
}
