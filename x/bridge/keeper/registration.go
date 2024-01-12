package keeper

// import (
// 	"context"
// 	"fmt"

// 	"github.com/tellor-io/layer/x/oracle/types"

// 	"github.com/cometbft/cometbft/libs/log"
// 	coretypes "github.com/cometbft/cometbft/rpc/core/types"
// 	"github.com/cosmos/cosmos-sdk/client"
// 	// gogogrpc "github.com/gogo/protobuf/grpc"
// 	// "github.com/grpc-ecosystem/grpc-gateway/runtime"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// )

// // to check queryServer implements ServiceServer
// var _ BridgeServer = bridgeServer{}

// // queryServer implements ServiceServer
// type bridgeServer struct {
// 	clientCtx client.Context
// }

// // NewQueryServer returns new queryServer from provided client.Context
// func NewQueryServer(clientCtx client.Context) BridgeServer {
// 	return bridgeServer{
// 		clientCtx: clientCtx,
// 	}
// }

// // func RegisterHeaderService(clientCtx client.Context, server gogogrpc.Server) {
// // 	RegisterBridgeServer(server, NewQueryServer(clientCtx))
// // }

// // // RegisterGRPCGatewayRoutes mounts the node gRPC service's GRPC-gateway routes
// // // on the given mux object.
// // func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
// // 	RegisterBridgeHandlerClient(context.Background(), mux, NewBridgeClient(clientConn))
// // }

// func (s bridgeServer) getCommit(height int64) (*coretypes.ResultCommit, error) {
// 	node, err := s.clientCtx.GetNode()
// 	if err != nil {
// 		return nil, err
// 	}
// 	var h *int64

// 	if height != 0 {
// 		h = &height
// 	}
// 	commit, err := node.Commit(context.Background(), h)
// 	return commit, err
// }

// func (s bridgeServer) Logger(ctx sdk.Context) log.Logger {
// 	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
// }

// // package bridge

// // import (
// // 	"context"

// // 	"github.com/cometbft/cometbft/crypto/merkle"
// // 	"github.com/ethereum/go-ethereum/common"
// // )

// // ChainID returns QueryChainIDResponse that has chain id from ctx
// // func (s bridgeServer) BlockheaderMerkleEVM(ctx context.Context, req *QueryBlockheaderMerkleRequest) (*QueryBlockheaderMerkleEVMResponse, error) {
// // commit, err := s.getCommit(ctx, req.Height)
// // if err != nil {
// // 	panic(err)
// // }

// // hbz, err := commit.Header.Version.Marshal()
// // if err != nil {
// // 	panic(err)
// // }
// // protobufBlockId := commit.Header.LastBlockID.ToProto()
// // bytesBlockId, err := protobufBlockId.Marshal()
// // if err != nil {
// // 	panic(err)
// // }
// // headerEvm := BlockHeaderMerkleEVM{
// // 	VersionChainidHash: common.BytesToHash(merkle.HashFromByteSlices([][]byte{
// // 		hbz,
// // 		cdcEncode(commit.Header.ChainID),
// // 	})).String(),
// // 	Height:         uint64(commit.Header.Height),
// // 	TimeSecond:     uint64(commit.Header.Time.Unix()),
// // 	TimeNanosecond: uint32(commit.Header.Time.Nanosecond()),
// // 	LastblockidCommitHash: common.BytesToHash(merkle.HashFromByteSlices([][]byte{
// // 		bytesBlockId,
// // 		cdcEncode(commit.Header.LastCommitHash),
// // 		cdcEncode(commit.Header.DataHash),
// // 		cdcEncode(commit.Header.ValidatorsHash),
// // 	})).String(),
// // 	NextvalidatorConsensusHash: common.BytesToHash(merkle.HashFromByteSlices([][]byte{
// // 		cdcEncode(commit.Header.NextValidatorsHash),
// // 		cdcEncode(commit.Header.ConsensusHash),
// // 	})).String(),
// // 	LastresultsHash: common.BytesToHash(merkle.HashFromByteSlices([][]byte{
// // 		cdcEncode(commit.Header.LastResultsHash),
// // 	})).String(),
// // 	EvidenceProposerHash: common.BytesToHash(merkle.HashFromByteSlices([][]byte{
// // 		cdcEncode(commit.Header.EvidenceHash),
// // 		cdcEncode(commit.Header.ProposerAddress),
// // 	})).String(),
// // }
// // 	return &QueryBlockheaderMerkleEVMResponse{
// // 		BlockheaderMerkleEvm: headerEvm,
// // 	}, nil
// // }
