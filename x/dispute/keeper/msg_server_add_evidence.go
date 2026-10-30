package keeper

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AddEvidence adds evidence to an existing dispute. If any of the evidence reports are an aggregate, flag them
func (k msgServer) AddEvidence(goCtx context.Context, msg *types.MsgAddEvidence) (*types.MsgAddEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get dispute by disputeId
	dispute, err := k.Disputes.Get(ctx, msg.DisputeId)
	if err != nil {
		return nil, err
	}
	// check if dispute is open
	if !dispute.Open {
		return nil, errors.New("dispute is not open")
	}
	// reporter in additional evidence must be the same as the reporter in the dispute
	for _, report := range msg.Reports {
		if !strings.EqualFold(report.Reporter, dispute.InitialEvidence.Reporter) {
			return nil, errors.New("reporter in additional evidence must be the same as the reporter in the dispute")
		}
	}
	// additional evidence must be less than 21 days old
	for _, report := range msg.Reports {
		if report.Timestamp.Before(ctx.BlockTime().Add(-21 * 24 * time.Hour)) {
			return nil, errors.New("additional evidence must be less than 21 days old")
		}
	}
	// enforce category-specific aggregate evidence limits before mutating state
	if err := k.validateAggregateEvidenceLimits(ctx, dispute, msg.Reports); err != nil {
		return nil, err
	}
	// append submitted evidence to dispute
	dispute.AdditionalEvidence = append(dispute.AdditionalEvidence, msg.Reports...)
	// set updated dispute
	err = k.Disputes.Set(ctx, msg.DisputeId, dispute)
	if err != nil {
		return nil, err
	}

	// for each microreport evidence, if the reporter is the aggregate reporter, flag it
	for _, report := range msg.Reports {
		err := k.oracleKeeper.FlagAggregateReport(ctx, *report)
		// if error is not nil and not collections.ErrNotFound, return error
		if err != nil && !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
	}
	return &types.MsgAddEvidenceResponse{}, nil
}

// validateAggregateEvidenceLimits enforces:
//   - Warning: no aggregate reports may be added via MsgAddEvidence
//   - Minor: at most one aggregate report may be flagged across initial + additional evidence
func (k msgServer) validateAggregateEvidenceLimits(ctx sdk.Context, dispute types.Dispute, reports []*oracletypes.MicroReport) error {
	switch dispute.DisputeCategory {
	case types.Warning:
		for _, report := range reports {
			isAggregate, err := k.oracleKeeper.IsAggregateReport(ctx, *report)
			if err != nil {
				return err
			}
			if isAggregate {
				return types.ErrWarningAggregateEvidence
			}
		}
	case types.Minor:
		aggregateCount := 0
		isAggregate, err := k.oracleKeeper.IsAggregateReport(ctx, dispute.InitialEvidence)
		if err != nil {
			return err
		}
		if isAggregate {
			aggregateCount++
		}
		for _, report := range dispute.AdditionalEvidence {
			isAggregate, err := k.oracleKeeper.IsAggregateReport(ctx, *report)
			if err != nil {
				return err
			}
			if isAggregate {
				aggregateCount++
			}
		}
		for _, report := range reports {
			isAggregate, err := k.oracleKeeper.IsAggregateReport(ctx, *report)
			if err != nil {
				return err
			}
			if isAggregate {
				aggregateCount++
			}
		}
		if aggregateCount > 1 {
			return types.ErrMinorAggregateEvidenceLimit
		}
	}
	return nil
}
