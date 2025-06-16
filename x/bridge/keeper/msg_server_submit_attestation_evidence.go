package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
)

// Submit evidence for a malicious oracle attestation.
func (k msgServer) SubmitAttestationEvidence(goCtx context.Context, msg *types.MsgSubmitAttestationEvidence) (*types.MsgSubmitAttestationEvidenceResponse, error) {
	err := k.Keeper.CheckAttestationEvidence(goCtx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitAttestationEvidenceResponse{}, nil
}
