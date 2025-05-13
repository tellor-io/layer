package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
)

// Update the number of attestation requests per block through governance.
func (k msgServer) SubmitAttestationEvidence(goCtx context.Context, msg *types.MsgSubmitAttestationEvidence) (*types.MsgSubmitAttestationEvidenceResponse, error) {
	err := k.Keeper.CheckAttestationEvidence(goCtx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitAttestationEvidenceResponse{}, nil
}
