package app_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"
	"github.com/tellor-io/layer/testutil/sample"

	"cosmossdk.io/log"

	baseappmock "github.com/cosmos/cosmos-sdk/baseapp/testutil/mock"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type ProposalHandlerTestSuite struct {
	suite.Suite
	ctx             sdk.Context
	proposalHandler *app.ProposalHandler
	oracleKeeper    *mocks.OracleKeeper
	bridgeKeeper    *mocks.BridgeKeeper
	stakingKeeper   *mocks.StakingKeeper
}

func (s *ProposalHandlerTestSuite) SetupTest() {
	// require := s.Require()
	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", os.TempDir())
	viper.Set("key-name", "my-key-name")

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	ok := mocks.NewOracleKeeper(s.T())
	bk := mocks.NewBridgeKeeper(s.T())
	sk := mocks.NewStakingKeeper(s.T())

	s.bridgeKeeper = bk
	s.oracleKeeper = ok
	s.stakingKeeper = sk

	s.ctx = testutils.CreateTestContext(s.T())
	ctrl := gomock.NewController(s.T())
	valStore := baseappmock.NewMockValidatorStore(ctrl)
	s.proposalHandler = app.NewProposalHandler(
		log.NewNopLogger(),
		valStore,
		cdc,
		ok,
		bk,
		sk,
	)
}

func TestProposalHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ProposalHandlerTestSuite))
}

func (s *ProposalHandlerTestSuite) TestValidatorOperatorAddressFromVote() {
	require := s.Require()
	p := s.proposalHandler
	require.NotNil(p)
	sk := s.stakingKeeper
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	valAddr := sample.AccAddressBytes()
	consAddr := sdk.ConsAddress(valAddr)
	vote := abcitypes.ExtendedVoteInfo{
		Validator: abcitypes.Validator{
			Address: consAddr,
		},
	}

	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)

	operatorAddr, err := p.ValidatorOperatorAddressFromVote(ctx, vote)
	require.NoError(err)
	require.Equal(valAddr.String(), operatorAddr)
}

func (s *ProposalHandlerTestSuite) TestCheckOracleAttestationsFromLastCommit() {
	require := s.Require()
	p := s.proposalHandler
	require.NotNil(p)
	sk := s.stakingKeeper
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	commit, voteExt, _, valAddr, consAddr := testutils.GenerateCommit(s.T(), ctx)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)

	att, snap, addrs, err := p.CheckOracleAttestationsFromLastCommit(ctx, commit)
	require.NoError(err)
	require.Equal(voteExt.OracleAttestations[0].Attestation, att[0])
	require.Equal(voteExt.OracleAttestations[0].Snapshot, snap[0])
	require.Equal(valAddr.String(), addrs[0])
	fmt.Println("addrs: ", addrs, "\natt: ", att, "\nsnap: ", snap)
}

func (s *ProposalHandlerTestSuite) TestSetEVMAddresses() {
	require := s.Require()
	p := s.proposalHandler
	require.NotNil(p)
	bk := s.bridgeKeeper
	require.NotNil(bk)
	ctx := s.ctx
	require.NotNil(ctx)

	operAddrs := []string{
		"0x1",
		"0x2",
		"0x3",
	}
	evmAddrs := []string{
		"0x1",
		"0x2",
		"0x3",
	}
	bk.On("SetEVMAddressByOperator", ctx, operAddrs[0], common.HexToAddress(evmAddrs[0]).Bytes()).Return(nil).Once()
	bk.On("SetEVMAddressByOperator", ctx, operAddrs[1], common.HexToAddress(evmAddrs[1]).Bytes()).Return(nil).Once()
	bk.On("SetEVMAddressByOperator", ctx, operAddrs[2], common.HexToAddress(evmAddrs[2]).Bytes()).Return(nil).Once()
	require.NoError(p.SetEVMAddresses(ctx, operAddrs, evmAddrs))

	bk.On("SetEVMAddressByOperator", ctx, operAddrs[0], common.HexToAddress(evmAddrs[0]).Bytes()).Return(errors.New("error")).Once()
	require.Error(p.SetEVMAddresses(ctx, operAddrs, evmAddrs))
}

func (s *ProposalHandlerTestSuite) TestCheckInitialSignaturesFromLastCommit() {
	require := s.Require()
	p := s.proposalHandler
	bk := s.bridgeKeeper
	sk := s.stakingKeeper
	require.NotNil(bk)
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	// not BlockIDFlagCommit vote
	valAccAddr1 := sample.AccAddressBytes()
	consAddr := sdk.ConsAddress(valAccAddr1)
	val1 := abcitypes.Validator{
		Address: consAddr,
	}
	ext := abcitypes.ExtendedCommitInfo{}
	ext.Votes = []abcitypes.ExtendedVoteInfo{
		{
			Validator: val1,
		},
	}
	res1, res2, err := p.CheckInitialSignaturesFromLastCommit(s.ctx, ext)
	require.NoError(err)
	require.Empty(res1)
	require.Empty(res2)

	// BlockIDFlagCommit vote
	commit, voteExt, addrsExpected, valAddr, consAddr := testutils.GenerateCommit(s.T(), ctx)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)
	bk.On("EVMAddressFromSignatures", ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB).Return(addrsExpected, nil)
	bk.On("GetEVMAddressByOperator", ctx, valAddr.String()).Return(nil, errors.New("error"))
	res1, res2, err = p.CheckInitialSignaturesFromLastCommit(ctx, commit)
	require.NoError(err)
	require.Equal(valAddr.String(), res1[0])
	require.Equal(addrsExpected.String(), res2[0])
}

func (s *ProposalHandlerTestSuite) TestCheckValsetSignaturesFromLastCommit() {
	require := s.Require()
	p := s.proposalHandler
	bk := s.bridgeKeeper
	sk := s.stakingKeeper
	require.NotNil(bk)
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	commit, voteExt, _, valAddr, consAddr := testutils.GenerateCommit(s.T(), ctx)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)

	operAddrs, timestamps, signatures, err := p.CheckValsetSignaturesFromLastCommit(ctx, commit)
	require.NoError(err)
	require.Equal(valAddr.String(), operAddrs[0])
	require.Equal(uint64(timestamps[0]), voteExt.ValsetSignature.Timestamp)
	require.Equal(signatures[0], hex.EncodeToString(voteExt.ValsetSignature.Signature))
}

func (s *ProposalHandlerTestSuite) TestPreBlocker() {
	require := s.Require()
	// p := s.proposalHandler
	bk := s.bridgeKeeper
	sk := s.stakingKeeper
	require.NotNil(bk)
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	// accAddr := sample.AccAddressBytes()
	// _ = sdk.ConsAddress(accAddr)
	// sigA, sigB, _ := testutils.GenerateSignatures(s.T())

	// voteExt := app.BridgeVoteExtension{
	// 	OracleAttestations: []app.OracleAttestation{
	// 		{
	// 			Attestation: []byte("attestation"),
	// 			Snapshot:    []byte("snapshot"),
	// 		},
	// 	},
	// 	InitialSignature: app.InitialSignature{
	// 		SignatureA: sigA,
	// 		SignatureB: sigB,
	// 	},
	// 	ValsetSignature: app.BridgeValsetSignature{
	// 		Signature: []byte("signature"),
	// 		Timestamp: uint64(ctx.BlockTime().Unix()),
	// 	},
	// }

	// voteExtBytes, err := json.Marshal(voteExt)
	// require.NoError(err)

	// req := abcitypes.RequestFinalizeBlock{
	// 	Txs: [][]byte{
	// 		voteExtBytes,
	// 	},
	// 	Height: 1,
	// }

	// res, err := p.PreBlocker(ctx, &req)
	// fmt.Println("res: ", res, "\nerr: ", err)
}

func (s *ProposalHandlerTestSuite) TestPrepareProposalHandler() {
	require := s.Require()
	p := s.proposalHandler
	require.NotNil(p)
	bk := s.bridgeKeeper
	sk := s.stakingKeeper
	require.NotNil(bk)
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	commit, _, _, _, _ := testutils.GenerateCommit(s.T(), ctx)
	ctx = ctx.WithBlockHeight(2)
	req := abcitypes.RequestPrepareProposal{
		Height:          2,
		LocalLastCommit: commit,
	}
	res, err := p.PrepareProposalHandler(ctx, &req)
	fmt.Println("res: ", res, "\nerr: ", err)
}
