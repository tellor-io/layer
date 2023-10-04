package bridge

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

// ChainID returns QueryChainIDResponse that has chain id from ctx
func (s bridgeServer) BlockheaderMerkleEVM(ctx context.Context, req *QueryBlockheaderMerkleRequest) (*QueryBlockheaderMerkleEVMResponse, error) {
	header, err := s.BlockheaderMerkle(ctx, req)
	if err != nil {
		return nil, err
	}
	headerEvm := BlockHeaderMerkleEvm{
		VersionChainidHash:         common.BytesToHash(header.BlockheaderMerkle.VersionChainidHash).String(),
		Height:                     header.BlockheaderMerkle.Height,
		TimeSecond:                 header.BlockheaderMerkle.TimeSecond,
		TimeNanosecond:             header.BlockheaderMerkle.TimeNanosecond,
		LastblockidCommitHash:      common.BytesToHash(header.BlockheaderMerkle.LastblockidCommitHash).String(),
		NextvalidatorConsensusHash: common.BytesToHash(header.BlockheaderMerkle.NextvalidatorConsensusHash).String(),
		LastresultsHash:            common.BytesToHash(header.BlockheaderMerkle.LastresultsHash).String(),
		EvidenceProposerHash:       common.BytesToHash(header.BlockheaderMerkle.EvidenceProposerHash).String(),
	}
	return &QueryBlockheaderMerkleEVMResponse{
		BlockheaderMerkleEvm: headerEvm,
	}, nil
}
