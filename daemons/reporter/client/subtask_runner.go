package client

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
)

type SubTaskRunner interface {
	RunReporterDaemonTaskLoop(
		ctx context.Context,
		client *Client,
		cosmosClient client.Context,
	) error
}

type SubTaskRunnerImpl struct{}

// Ensure SubTaskRunnerImpl implements the SubTaskRunner interface.
var _ SubTaskRunner = (*SubTaskRunnerImpl)(nil)

func (s *SubTaskRunnerImpl) RunReporterDaemonTaskLoop(
	ctx context.Context,
	daemonClient *Client,
	cosmosClient client.Context,
) error {
	err := daemonClient.SubmitReport(ctx)
	if err != nil {
		return err
	}
	return nil
}
