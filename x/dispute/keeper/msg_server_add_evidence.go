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
	// evidence is only accepted once the dispute fee is fully paid
	if dispute.DisputeStatus == types.Prevote {
		return nil, types.ErrDisputeFeeNotPaid
	}
	disputedReporter, err := sdk.AccAddressFromBech32(dispute.InitialEvidence.Reporter)
	if err != nil {
		return nil, err
	}
	// use the reports stored on chain, not the copies the caller sent
	reports := make([]*oracletypes.MicroReport, 0, len(msg.Reports))
	for _, report := range msg.Reports {
		// reporter in additional evidence must be the same as the reporter in the dispute
		if !strings.EqualFold(report.Reporter, dispute.InitialEvidence.Reporter) {
			return nil, errors.New("reporter in additional evidence must be the same as the reporter in the dispute")
		}
		stored, exists, err := k.oracleKeeper.ValidateMicroReportExists(ctx, disputedReporter, report.MetaId, report.QueryId)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, types.ErrEvidenceReportDoesNotExist
		}
		// additional evidence must be less than 21 days old
		if stored.Timestamp.Before(ctx.BlockTime().Add(-21 * 24 * time.Hour)) {
			return nil, errors.New("additional evidence must be less than 21 days old")
		}
		reports = append(reports, stored)
	}
	// enforce category-specific aggregate evidence limits before mutating state
	if err := k.validateAggregateEvidenceLimits(ctx, dispute, reports); err != nil {
		return nil, err
	}
	// append submitted evidence to dispute
	dispute.AdditionalEvidence = append(dispute.AdditionalEvidence, reports...)
	// set updated dispute
	err = k.Disputes.Set(ctx, msg.DisputeId, dispute)
	if err != nil {
		return nil, err
	}

	// for each microreport evidence, if the reporter is the aggregate reporter, flag it
	for _, report := range reports {
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
