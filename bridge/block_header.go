package bridge

import (
	"context"

	"github.com/cometbft/cometbft/crypto/merkle"
)

// ChainID returns QueryChainIDResponse that has chain id from ctx
func (s bridgeServer) BlockheaderMerkle(_ context.Context, req *QueryBlockheaderMerkleRequest) (*QueryBlockheaderMerkleResponse, error) {
	commit, err := s.getCommit(req.Height)
	if err != nil {
		panic(err)
	}

	hbz, err := commit.Header.Version.Marshal()
	if err != nil {
		panic(err)
	}
	protobufBlockId := commit.Header.LastBlockID.ToProto()
	bytesBlockId, err := protobufBlockId.Marshal()
	if err != nil {
		panic(err)
	}
	var header BlockHeaderMerkle
	header = BlockHeaderMerkle{
		VersionChainidHash: merkle.HashFromByteSlices([][]byte{
			hbz,
			cdcEncode(commit.Header.ChainID),
		}),
		Height:         uint64(commit.Header.Height),
		TimeSecond:     uint64(commit.Header.Time.Unix()),
		TimeNanosecond: uint32(commit.Header.Time.Nanosecond()),
		LastblockidCommitHash: merkle.HashFromByteSlices([][]byte{
			bytesBlockId,
			cdcEncode(commit.Header.LastCommitHash),
			cdcEncode(commit.Header.DataHash),
			cdcEncode(commit.Header.ValidatorsHash),
		}),
		NextvalidatorConsensusHash: merkle.HashFromByteSlices([][]byte{
			cdcEncode(commit.Header.NextValidatorsHash),
			cdcEncode(commit.Header.ConsensusHash),
		}),
		LastresultsHash: merkle.HashFromByteSlices([][]byte{
			cdcEncode(commit.Header.LastResultsHash),
		}),
		EvidenceProposerHash: merkle.HashFromByteSlices([][]byte{
			cdcEncode(commit.Header.EvidenceHash),
			cdcEncode(commit.Header.ProposerAddress),
		}),
	}
	return &QueryBlockheaderMerkleResponse{
		BlockheaderMerkle: header,
	}, nil
}
