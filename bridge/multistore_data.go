package bridge

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	fmt "fmt"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"github.com/cometbft/cometbft/libs/bytes"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	ics23 "github.com/confio/ics23/go"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func hashQdata(queryData string) ([]byte, error) {
	// Decode the hex-encoded input string
	qbytes, err := hex.DecodeString(queryData)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(qbytes), nil
}

func (s bridgeServer) MultistoreTree(ctx context.Context, req *QueryMultistoreRequest) (*QueryMultistoreResponse, error) {
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
	for i, p := range paths {
		evmdata[i].IsDataOnRight = p.IsDataOnRight
		evmdata[i].SubtreeHeight = p.SubtreeHeight
		evmdata[i].SubtreeSize = int64(p.SubtreeSize)
		evmdata[i].SubtreeVersion = int64(p.SubtreeVersion)
		evmdata[i].SiblingHash = bytes.HexBytes(p.SiblingHash).String()
	}

	return &QueryMultistoreResponse{
		MutiStoreTree: MutiStoreTreeFields{
			LuqchainIavlStateHash:            bytes.HexBytes(multistoreProof.Value).String(),
			MintStoreMerkleHash:              bytes.HexBytes(multistoreProof.Path[0].Suffix).String(),
			IcacontrollerToIcahostMerkleHash: bytes.HexBytes(multistoreProof.Path[1].Prefix[1:]).String(),
			FeegrantToIbcMerkleHash:          bytes.HexBytes(multistoreProof.Path[2].Prefix[1:]).String(),
			AccToEvidenceMerkleHash:          bytes.HexBytes(multistoreProof.Path[3].Prefix[1:]).String(),
			ParamsToVestingMerkleHash:        bytes.HexBytes(multistoreProof.Path[4].Suffix).String(),
		},
		Iavl: evmdata,
	}, nil
}

func GetMerklePaths(iavlEp *ics23.ExistenceProof) []IAVLMerklePath {
	paths := make([]IAVLMerklePath, 0)
	for _, step := range iavlEp.Path {
		if step.Hash != ics23.HashOp_SHA256 {
			// Tendermint v0.34.9 is using SHA256 only.
			panic("Expect HashOp_SHA256")
		}
		imp := IAVLMerklePath{}

		// decode IAVL inner prefix
		// ref: https://github.com/cosmos/iavl/blob/master/proof_ics23.go#L96
		subtreeHeight, n1 := binary.Varint(step.Prefix)
		subtreeSize, n2 := binary.Varint(step.Prefix[n1:])
		subtreeVersion, n3 := binary.Varint(step.Prefix[n1+n2:])

		imp.SubtreeHeight = uint32(subtreeHeight)
		imp.SubtreeSize = uint64(subtreeSize)
		imp.SubtreeVersion = uint64(subtreeVersion)

		prefixLength := n1 + n2 + n3 + 1
		if prefixLength != len(step.Prefix) {
			imp.IsDataOnRight = true
			imp.SiblingHash = step.Prefix[prefixLength : len(step.Prefix)-1] // remove 0x20
		} else {
			imp.IsDataOnRight = false
			imp.SiblingHash = step.Suffix[1:] // remove 0x20
		}
		paths = append(paths, imp)
	}
	return paths
}

// func encodeToEthFormat(merklePath IAVLMerklePath) IAVLMerklePathEvm {
// 	return IAVLMerklePathEvm{
// 		merklePath.IsDataOnRight,
// 		merklePath.SubtreeHeight,
// 		int64(merklePath.SubtreeSize),
// 		int64(merklePath.SubtreeVersion),
// 		bytes.HexBytes(merklePath.SiblingHash).String(),
// 	}
// }
