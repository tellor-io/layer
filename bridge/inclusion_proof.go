package bridge

import (
	"context"
	"crypto/sha256"
	fmt "fmt"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"github.com/cometbft/cometbft/libs/bytes"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	ics23 "github.com/confio/ics23/go"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s bridgeServer) InclusionProof(ctx context.Context, req *QueryInclusionProofRequest) (*QueryInclusionProofResponse, error) {
	var h *int64
	if req.Height != 0 {
		h = &req.Height
	}
	qid, err := hashQdata(req.Qid)
	if err != nil {
		return nil, err
	}
	tbytes := keeper.Uint64ToBytes(req.Timestamp)
	resp, err := s.clientCtx.Client.ABCIQueryWithOptions(
		context.Background(),
		"/store/luqchain/key",
		append(types.KeyPrefix(types.ReportsKey), append(qid, tbytes...)...),
		cometclient.ABCIQueryOptions{Height: *h, Prove: true},
	)
	if err != nil {
		return nil, err
	}
	proof := resp.Response.GetProofOps()
	if proof == nil {
		return nil, nil
	}
	ops := proof.GetOps()
	if ops == nil {
		return nil, nil
	}

	var multistoreProof *ics23.ExistenceProof
	var iavlProof *ics23.ExistenceProof

	for _, op := range ops {
		switch op.GetType() {
		case storetypes.ProofOpIAVLCommitment:
			proof := &ics23.CommitmentProof{}
			err := proof.Unmarshal(op.Data)
			if err != nil {
				panic(err)
			}
			iavlCOps := storetypes.NewIavlCommitmentOp(op.Key, proof)
			iavlProof = iavlCOps.Proof.GetExist()
			if iavlProof == nil {
				return nil, nil
			}
		case storetypes.ProofOpSimpleMerkleCommitment:
			proof := &ics23.CommitmentProof{}
			err := proof.Unmarshal(op.Data)
			if err != nil {
				panic(err)
			}
			multiStoreOps := storetypes.NewSimpleMerkleCommitmentOp(op.Key, proof)
			multistoreProof = multiStoreOps.Proof.GetExist()
			if multistoreProof == nil {
				return nil, nil
			}
			appHash, err := multistoreProof.Calculate()
			fmt.Println("appHash", bytes.HexBytes(appHash))
			if err != nil {
				fmt.Println("err", err)
			}

		default:
			fmt.Println("Defaulting to nothing found")
			return nil, nil
		}
	}
	logCtx := sdk.UnwrapSDKContext(ctx)
	s.Logger(logCtx).Error(fmt.Sprintf("iavlProof %v", iavlProof))
	paths := GetMerklePaths(iavlProof)
	evmdata := make([]IAVLMerklePathEvm, len(paths))
	s.Logger(logCtx).Error(fmt.Sprintf("paths %v", paths)) // todo: fix or rmv
	// log the iavl proof value
	s.Logger(logCtx).Error(fmt.Sprintf("iavlProof.Value %v", iavlProof.Value))
	for i, p := range paths {
		evmdata[i].IsDataOnRight = p.IsDataOnRight
		evmdata[i].SubtreeHeight = p.SubtreeHeight
		evmdata[i].SubtreeSize = int64(p.SubtreeSize)
		evmdata[i].SubtreeVersion = int64(p.SubtreeVersion)
		evmdata[i].SiblingHash = bytes.HexBytes(p.SiblingHash).String()
	}
	hash := sha256.Sum256(iavlProof.Value)
	dataHash := bytes.HexBytes(hash[:]).String()

	return &QueryInclusionProofResponse{
		InclusionProofStuff: InclusionProofStuffFields{
			RootHash: bytes.HexBytes(multistoreProof.Value).String(),
			Version:  int64(req.Height),
			Key:      bytes.HexBytes(iavlProof.Key).String(),
			DataHash: dataHash,
		},
		MerklePath: evmdata,
	}, nil
}
