package bridge

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	fmt "fmt"
	"log"

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
	resp, err := s.clientCtx.Client.ABCIQueryWithOptions(
		context.Background(),
		"/store/oracle/key",
		bytes.HexBytes(types.KeyPrefix(types.ReportsKey)),
		cometclient.ABCIQueryOptions{Prove: true},
	)
	if err != nil {
		return nil, err
	}
	log.Printf("here0")
	proof := resp.Response.GetProofOps()
	if proof == nil {
		return nil, nil
	}
	log.Printf("here1")
	ops := proof.GetOps()
	if ops == nil {
		return nil, nil
	}

	var multistoreProof *ics23.ExistenceProof
	var iavlProof *ics23.ExistenceProof

	for _, op := range ops {
		switch op.GetType() {
		case storetypes.ProofOpSimpleMerkleCommitment:
			proof := &ics23.CommitmentProof{}
			err := proof.Unmarshal(op.Data)
			if err != nil {
				panic(err)
			}
			multiStoreOps := storetypes.NewSimpleMerkleCommitmentOp(op.Key, proof)
			multistoreProof = multiStoreOps.Proof.GetExist()
			appHash, err := multistoreProof.Calculate()
			fmt.Println("appHash", bytes.HexBytes(appHash))
			if err != nil {
				fmt.Println("err", err)
			}

		default:
			fmt.Println("Defaulting to nothing found")
		}
	}
	logCtx := sdk.UnwrapSDKContext(ctx)
	s.Logger(logCtx).Error(fmt.Sprintf("iavlProof %v", iavlProof))

	return &QueryMultistoreResponse{
		MutiStoreTree: MutiStoreTreeFields{
			OracleIavlStateHash:              bytes.HexBytes(multistoreProof.Value).String(),
			MintStoreMerkleHash:              bytes.HexBytes(multistoreProof.Path[0].Prefix).String(),
			IcacontrollerToIcahostMerkleHash: bytes.HexBytes(multistoreProof.Path[1].Prefix[1:]).String(),
			FeegrantToIbcMerkleHash:          bytes.HexBytes(multistoreProof.Path[2].Prefix[1:]).String(),
			AccToEvidenceMerkleHash:          bytes.HexBytes(multistoreProof.Path[3].Prefix[1:]).String(),
			ParamsToVestingMerkleHash:        bytes.HexBytes(multistoreProof.Path[4].Suffix).String(),
		},
		Iavl: nil,
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
