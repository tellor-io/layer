package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/dispute/types"

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
