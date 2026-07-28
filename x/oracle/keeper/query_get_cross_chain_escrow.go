package keeper

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
)

// GetCrossChainEscrow returns the escrow record and settlement queryId for an
// Ethereum escrow identified by (chain id, contract address, escrow id).
func (k Querier) GetCrossChainEscrow(ctx context.Context, req *types.QueryGetCrossChainEscrowRequest) (*types.QueryGetCrossChainEscrowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	contract, err := hex.DecodeString(req.EscrowContract)
	if err != nil || len(contract) != 20 {
		return nil, status.Error(codes.InvalidArgument, "escrow contract must be a hex-encoded 20-byte address")
	}

	escrowKey, _, err := CrossChainSettlementQueryId(req.EscrowChainId, contract, req.EscrowId)
	if err != nil {
		return nil, err
	}

	escrow, err := k.keeper.CrossChainEscrows.Get(ctx, escrowKey)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "escrow not found")
		}
		return nil, err
	}

	return &types.QueryGetCrossChainEscrowResponse{
		Escrow:            escrow,
		SettlementQueryId: hex.EncodeToString(escrowKey),
	}, nil
}
