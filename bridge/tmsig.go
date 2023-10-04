package bridge

import (
	"context"
	fmt "fmt"
	"sort"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cometbft/cometbft/libs/protoio"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cometbft "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	// 	"fmt"
	// 	"sort"
	// 	"strconv"
	// "github.com/cometbft/cometbft/crypto/secp256k1"
	// "github.com/cometbft/cometbft/crypto/tmhash"
	// "github.com/cometbft/cometbft/libs/protoio"
	// tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	// cometbft "github.com/cometbft/cometbft/types"
	// "github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// "github.com/ethereum/go-ethereum/crypto"
	// "github.com/spf13/cobra"
	// "github.com/cometbft/cometbft/libs/bytes"
)

// // BlockCommand returns the verified block data for a given heights
// func TendermintSignatures() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "tendermint-signatures",
// 		Short: "Get Signatures",
// 		Args:  cobra.MaximumNArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			clientCtx, err := client.GetClientQueryContext(cmd)
// 			if err != nil {
// 				return err
// 			}
// 			var height *int64

// 			// optional height
// 			if len(args) > 0 {
// 				h, err := strconv.Atoi(args[0])
// 				if err != nil {
// 					return err
// 				}
// 				if h > 0 {
// 					tmp := int64(h)
// 					height = &tmp
// 				}
// 			}

// 			output, err := fetchSig(clientCtx, height)
// 			if err != nil {
// 				return err
// 			}

// 			fmt.Println(string(output))
// 			return nil
// 		},
// 	}

// 	cmd.Flags().StringP(flags.FlagNode, "n", "tcp://localhost:26657", "Node to connect to")

// 	return cmd
// }

func (s bridgeServer) TmSig(_ context.Context, req *QueryTmRequest) (*QueryTmResponse, error) {
	commit, err := s.getCommit(req.Height)
	if err != nil {
		panic(err)
	}
	sigs, common, err := GetSignaturesAndPrefix(&commit.SignedHeader)
	if err != nil {
		panic(err)
	}
	return &QueryTmResponse{
		TmSig:  sigs,
		Common: common,
	}, nil
}

// func fetchSig(clientCtx client.Context, height *int64) ([]byte, error) {
// 	node, err := clientCtx.GetNode()
// 	if err != nil {
// 		return nil, err
// 	}
// 	commit, err := node.Commit(context.Background(), height)
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Println(&commit.SignedHeader)
// 	fmt.Println(GetSignaturesAndPrefix(&commit.SignedHeader))
// 	return nil, nil
// }

func recoverETHAddress(msg, sig, signer []byte) ([]byte, uint8, error) {
	for i := uint8(0); i < 2; i++ {
		pubuc, err := crypto.SigToPub(tmhash.Sum(msg), append(sig, byte(i)))
		if err != nil {
			return nil, 0, err
		}
		pub := crypto.CompressPubkey(pubuc)
		fmt.Println("pub", pub, "signer", signer)
		var tmp [33]byte

		copy(tmp[:], pub)

		if string(signer) == string(secp256k1.PubKey(tmp[:]).Address()) {
			return crypto.PubkeyToAddress(*pubuc).Bytes(), 27 + i, nil
		}
	}
	return nil, 0, fmt.Errorf("No match address found")
}

func GetPrefix(t tmproto.SignedMsgType, height int64, round int64) ([]byte, error) {
	prefix, err := protoio.MarshalDelimited(
		&tmproto.CanonicalVote{
			Type:   t,
			Height: height,
			Round:  round,
		},
	)
	if err != nil {
		return nil, err
	}
	length := int(prefix[0])
	// prefix should be X + default timestamp that equals to `2a0b088092b8c398feffffff01`, so we trim last 13 bytes
	return prefix[1 : length-12], nil
}

func GetSignaturesAndPrefix(info *cometbft.SignedHeader) ([]TmSig, CommonEncodedVotePart, error) {
	addrs := []string{}
	mapAddrs := map[string]TmSig{}

	prefix, err := GetPrefix(tmproto.SignedMsgType(info.Commit.Type()), info.Commit.Height, int64(info.Commit.Round))
	if err != nil {
		return nil, CommonEncodedVotePart{}, err
	}

	prefix = append(prefix, []byte{34, 72, 10, 32}...)

	suffix, err := protoio.MarshalDelimited(
		&tmproto.CanonicalPartSetHeader{
			Total: info.Commit.BlockID.PartSetHeader.Total,
			Hash:  info.Commit.BlockID.PartSetHeader.Hash,
		},
	)
	if err != nil {
		return nil, CommonEncodedVotePart{}, err
	}

	suffix = append([]byte{18}, suffix...)

	commonVote := CommonEncodedVotePart{SignedDataPrefix: prefix, SignedDataSuffix: suffix}

	commonPart := append(commonVote.SignedDataPrefix, info.Commit.BlockID.Hash...)
	commonPart = append(commonPart, commonVote.SignedDataSuffix...)

	chainIDBytes := []byte(info.ChainID)
	encodedChainIDConstant := append([]byte{50, uint8(len(chainIDBytes))}, chainIDBytes...)

	for _, vote := range info.Commit.Signatures {
		if !vote.ForBlock() {
			continue
		}

		encodedTimestamp := encodeTime(vote.Timestamp)

		msg := append(commonPart, []byte{42, uint8(len(encodedTimestamp))}...)
		msg = append(msg, encodedTimestamp...)
		msg = append(msg, encodedChainIDConstant...)
		msg = append([]byte{uint8(len(msg))}, msg...)

		addr, v, err := recoverETHAddress(msg, vote.Signature, vote.ValidatorAddress)

		if err != nil {
			return nil, CommonEncodedVotePart{}, err
		}
		addrs = append(addrs, string(addr))
		mapAddrs[string(addr)] = TmSig{
			common.BytesToHash(vote.Signature[:32]).Hex(),
			common.BytesToHash(vote.Signature[32:]).Hex(),
			uint32(v),
			common.BytesToHash(encodedTimestamp).Hex(),
		}
	}
	if len(addrs) == 0 {
		return nil, CommonEncodedVotePart{}, fmt.Errorf("No valid precommit")
	}

	signatures := make([]TmSig, len(addrs))
	sort.Strings(addrs)
	for i, addr := range addrs {
		signatures[i] = mapAddrs[addr]
	}

	return signatures, commonVote, nil
}
