package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
)

// Submit evidence for a malicious validator set update signature.
func (k msgServer) SubmitValsetSignatureEvidence(goCtx context.Context, msg *types.MsgSubmitValsetSignatureEvidence) (*types.MsgSubmitValsetSignatureEvidenceResponse, error) {
	err := k.Keeper.CheckValsetSignatureEvidence(goCtx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitValsetSignatureEvidenceResponse{}, nil
}
