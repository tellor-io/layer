package app_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"
	"github.com/tellor-io/layer/testutil/sample"
	"go.uber.org/mock/gomock"

	coreheader "cosmossdk.io/core/header"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
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
	valStore        *baseappmock.MockValidatorStore
}

func (s *ProposalHandlerTestSuite) SetupTest() {
	// require := s.Require()
	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", os.TempDir())
	viper.Set("key-name", "my-key-name")

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	s.oracleKeeper = mocks.NewOracleKeeper(s.T())
	s.bridgeKeeper = mocks.NewBridgeKeeper(s.T())
	s.stakingKeeper = mocks.NewStakingKeeper(s.T())

	s.ctx = testutils.CreateTestContext(s.T())
	ctrl := gomock.NewController(s.T())
	s.valStore = baseappmock.NewMockValidatorStore(ctrl)
	s.proposalHandler = app.NewProposalHandler(
		log.NewNopLogger(),
		s.valStore,
		cdc,
		s.oracleKeeper,
		s.bridgeKeeper,
		s.stakingKeeper,
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

	commit, voteExt, _, valAddr, consAddr, _ := testutils.GenerateCommit(s.T(), ctx)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)

	att, snap, addrs := p.CheckOracleAttestationsFromLastCommit(ctx, commit)
	require.Equal(voteExt.OracleAttestations[0].Attestation, att[0])
	require.Equal(voteExt.OracleAttestations[0].Snapshot, snap[0])
	require.Equal(valAddr.String(), addrs[0])
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
	}
	evmAddrs := []string{
		"0x1",
	}
	bk.On("SetEVMAddressByOperator", ctx, operAddrs[0], common.HexToAddress(evmAddrs[0]).Bytes()).Return(nil).Once()
	// bk.On("SetEVMAddressByOperator", ctx, operAddrs[1], common.HexToAddress(evmAddrs[1]).Bytes()).Return(nil).Once()
	// bk.On("SetEVMAddressByOpeator", ctx, operAddrs[2], common.HexToAddress(evmAddrs[2]).Bytes()).Return(nil).Once()
	require.NoError(p.SetEVMAddresses(ctx, operAddrs, evmAddrs))

	bk.On("SetEVMAddressByOperator", ctx, operAddrs[0], common.HexToAddress(evmAddrs[0]).Bytes()).Return(errors.New("error")).Once()
	require.NoError(p.SetEVMAddresses(ctx, operAddrs, evmAddrs))
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
	res1, res2 := p.CheckInitialSignaturesFromLastCommit(s.ctx, ext)
	require.Empty(res1)
	require.Empty(res2)

	// BlockIDFlagCommit vote
	commit, voteExt, addrsExpected, valAddr, consAddr, _ := testutils.GenerateCommit(s.T(), ctx)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)
	bk.On("EVMAddressFromSignatures", ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB, valAddr.String()).Return(addrsExpected, nil).Once()
	bk.On("GetEVMAddressByOperator", ctx, valAddr.String()).Return(nil, errors.New("error"))
	res1, res2 = p.CheckInitialSignaturesFromLastCommit(ctx, commit)
	require.Equal(valAddr.String(), res1[0])
	require.Equal(addrsExpected.String(), res2[0])

	bk.On("EVMAddressFromSignatures", ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB, valAddr.String()).Return(nil, errors.New("error")).Once()
	res1, res2 = p.CheckInitialSignaturesFromLastCommit(ctx, commit)
	require.Empty(res1)
	require.Empty(res2)
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

	commit, voteExt, _, valAddr, consAddr, _ := testutils.GenerateCommit(s.T(), ctx)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil)

	operAddrs, timestamps, signatures := p.CheckValsetSignaturesFromLastCommit(ctx, commit)
	require.Equal(valAddr.String(), operAddrs[0])
	require.Equal(uint64(timestamps[0]), voteExt.ValsetSignature.Timestamp)
	require.Equal(signatures[0], hex.EncodeToString(voteExt.ValsetSignature.Signature))
}

// func (s *ProposalHandlerTestSuite) TestPrepareProposalHandler() ([][]byte, sdk.AccAddress) {
// 	require := s.Require()
// 	p := s.proposalHandler
// 	bk := s.bridgeKeeper
// 	sk := s.stakingKeeper
// 	ctx := s.ctx
// 	require.NotNil(p)
// 	require.NotNil(bk)
// 	require.NotNil(sk)
// 	require.NotNil(ctx)

// 	extCommit, voteExt, evmAddr, accAddr, consAddr, _ := testutils.GenerateCommit(s.T(), ctx)

// 	// Set up mock expectations with gomock matchers
// 	s.valStore.EXPECT().GetPubKeyByConsAddr(gomock.Any(), consAddr).Return(cmtprotocrypto.PublicKey{}, nil).AnyTimes()
// 	bk.On("EVMAddressFromSignatures", ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB, consAddr.String()).Return(evmAddr, nil)
// 	bk.On("GetEVMAddressByOperator", ctx, accAddr.String()).Return(nil, errors.New("error"))

// 	lastCommit := abcitypes.CommitInfo{
// 		Round: 2,
// 		Votes: []abcitypes.VoteInfo{
// 			{
// 				Validator: abcitypes.Validator{
// 					Address: accAddr.Bytes(),
// 					Power:   1000,
// 				},
// 			},
// 		},
// 	}
// 	cometInfo := baseapp.NewBlockInfo(
// 		nil,
// 		nil,
// 		nil,
// 		lastCommit,
// 	)

// 	ctx = ctx.WithBlockHeight(3)
// 	ctx = ctx.WithCometInfo(cometInfo)
// 	ctx = ctx.WithHeaderInfo(coreheader.Info{
// 		Height: 3,
// 	})

// 	req := abcitypes.RequestPrepareProposal{
// 		Height:          3,
// 		LocalLastCommit: extCommit,
// 	}

// 	res, err := p.PrepareProposalHandler(ctx, &req)
// 	require.NoError(err)
// 	require.NotNil(res)

// 	return res.Txs, accAddr
// }

func (s *ProposalHandlerTestSuite) TestProcessProposalHandler() {
	require := s.Require()
	p := s.proposalHandler
	require.NotNil(p)
	ctx := s.ctx
	require.NotNil(ctx)
	bk := s.bridgeKeeper
	require.NotNil(bk)
	sk := s.stakingKeeper
	require.NotNil(sk)

	pubKey, privKey, consAddr, _ := testutils.GenerateProposer(s.T())
	sigA, sigB, evmAddr := testutils.GenerateSignatures(s.T())
	sk.On("GetParams", mock.Anything).Return(stakingtypes.Params{
		MaxValidators: 100,
	}, nil)

	ve := app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{
				Attestation: []byte("attestation"),
				Snapshot:    []byte("snapshot"),
			},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: sigA,
			SignatureB: sigB,
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: []byte("valSetSignature"),
			Timestamp: uint64(ctx.BlockTime().Unix()),
		},
	}
	veBz, err := json.Marshal(ve)
	require.NoError(err)

	_, _, sig, err := testutils.SignCVE(veBz, 2, 2, privKey)
	require.NoError(err)

	localLastCommit := abcitypes.ExtendedCommitInfo{
		Votes: []abcitypes.ExtendedVoteInfo{
			{
				Validator: abcitypes.Validator{
					Address: consAddr.Bytes(),
					Power:   1000000000000,
				},
				VoteExtension:      veBz,
				BlockIdFlag:        cmtproto.BlockIDFlagCommit,
				ExtensionSignature: sig,
			},
		},
		Round: 2,
	}

	opAndEVMAddrs := app.OperatorAndEVM{
		OperatorAddresses: []string{consAddr.String()},
		EVMAddresses:      []string{evmAddr.String()},
	}
	valsetSigs := app.ValsetSignatures{
		OperatorAddresses: []string{consAddr.String()},
		Timestamps:        []int64{int64(ve.ValsetSignature.Timestamp)},
		Signatures:        []string{hex.EncodeToString(ve.ValsetSignature.Signature)},
	}
	oracleAttestations := app.OracleAttestations{
		OperatorAddresses: []string{consAddr.String()},
		Attestations:      [][]byte{ve.OracleAttestations[0].Attestation},
		Snapshots:         [][]byte{ve.OracleAttestations[0].Snapshot},
	}

	injectedVoteExtTx := app.VoteExtTx{
		BlockHeight:        int64(2),
		OpAndEVMAddrs:      opAndEVMAddrs,
		ValsetSigs:         valsetSigs,
		OracleAttestations: oracleAttestations,
		ExtendedCommitInfo: localLastCommit,
	}
	injBz, err := json.Marshal(injectedVoteExtTx)
	require.NoError(err)

	lastCommit := abcitypes.CommitInfo{
		Round: 2,
		Votes: []abcitypes.VoteInfo{
			{
				Validator: abcitypes.Validator{
					Address: consAddr.Bytes(),
					Power:   1000000000000,
				},
				BlockIdFlag: cmtproto.BlockIDFlagCommit,
			},
		},
	}
	cometInfo := baseapp.NewBlockInfo(
		nil,
		nil,
		nil,
		lastCommit,
	)

	ctx = ctx.WithBlockHeight(3)
	ctx = ctx.WithCometInfo(cometInfo)
	ctx = ctx.WithHeaderInfo(coreheader.Info{
		Height:  3,
		ChainID: "layer",
	})

	req := abcitypes.RequestProcessProposal{
		Txs:                [][]byte{injBz},
		Height:             3,
		ProposedLastCommit: lastCommit,
	}

	validPubKey := cmtprotocrypto.PublicKey{
		Sum: &cmtprotocrypto.PublicKey_Ed25519{
			Ed25519: pubKey,
		},
	}
	s.valStore.EXPECT().GetPubKeyByConsAddr(ctx, consAddr).Return(validPubKey, nil).AnyTimes()

	bk.On("EVMAddressFromSignatures", ctx, sigA, sigB, consAddr.String()).Return(evmAddr, nil)
	bk.On("GetEVMAddressByOperator", ctx, consAddr.String()).Return(nil, errors.New("error"))
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: consAddr.String(),
	}, nil)
	res, err := p.ProcessProposalHandler(ctx, &req)
	require.NoError(err)
	require.NotNil(res)
}

func (s *ProposalHandlerTestSuite) TestPreBlocker() {
	require := s.Require()
	p := s.proposalHandler
	bk := s.bridgeKeeper
	sk := s.stakingKeeper
	require.NotNil(bk)
	require.NotNil(sk)
	ctx := s.ctx
	require.NotNil(ctx)

	_, privKey, consAddr, _ := testutils.GenerateProposer(s.T())
	sigA, sigB, evmAddr := testutils.GenerateSignatures(s.T())

	ve := app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{
				Attestation: []byte("attestation"),
				Snapshot:    []byte("snapshot"),
			},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: sigA,
			SignatureB: sigB,
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: []byte("valSetSignature"),
			Timestamp: uint64(ctx.BlockTime().Unix()),
		},
	}
	veBz, err := json.Marshal(ve)
	require.NoError(err)

	_, _, sig, err := testutils.SignCVE(veBz, 2, 2, privKey)
	require.NoError(err)

	localLastCommit := abcitypes.ExtendedCommitInfo{
		Votes: []abcitypes.ExtendedVoteInfo{
			{
				Validator: abcitypes.Validator{
					Address: consAddr.Bytes(),
					Power:   1000000000000,
				},
				VoteExtension:      veBz,
				BlockIdFlag:        cmtproto.BlockIDFlagCommit,
				ExtensionSignature: sig,
			},
		},
		Round: 2,
	}

	opAndEVMAddrs := app.OperatorAndEVM{
		OperatorAddresses: []string{consAddr.String()},
		EVMAddresses:      []string{evmAddr.String()},
	}
	valsetSigs := app.ValsetSignatures{
		OperatorAddresses: []string{consAddr.String()},
		Timestamps:        []int64{int64(ve.ValsetSignature.Timestamp)},
		Signatures:        []string{hex.EncodeToString(ve.ValsetSignature.Signature)},
	}
	oracleAttestations := app.OracleAttestations{
		OperatorAddresses: []string{consAddr.String()},
		Attestations:      [][]byte{ve.OracleAttestations[0].Attestation},
		Snapshots:         [][]byte{ve.OracleAttestations[0].Snapshot},
	}

	injectedVoteExtTx := app.VoteExtTx{
		BlockHeight:        int64(2),
		OpAndEVMAddrs:      opAndEVMAddrs,
		ValsetSigs:         valsetSigs,
		OracleAttestations: oracleAttestations,
		ExtendedCommitInfo: localLastCommit,
	}
	injBz, err := json.Marshal(injectedVoteExtTx)
	require.NoError(err)

	req := abcitypes.RequestFinalizeBlock{
		Txs:    [][]byte{injBz},
		Height: 3,
	}

	ctx = ctx.WithBlockHeight(3)

	var veTx app.VoteExtTx
	err = json.Unmarshal(injBz, &veTx)
	require.NoError(err)

	bk.On("EVMAddressFromSignatures", ctx, sigA, sigB, consAddr.String()).Return(evmAddr, nil)
	sk.On("GetValidatorByConsAddr", ctx, consAddr).Return(stakingtypes.Validator{
		OperatorAddress: consAddr.String(),
	}, nil)
	bk.On("SetEVMAddressByOperator", ctx, consAddr.String(), evmAddr.Bytes()).Return(nil)
	bk.On("SetBridgeValsetSignature", ctx, consAddr.String(), ve.ValsetSignature.Timestamp, veTx.ValsetSigs.Signatures[0]).Return(nil)
	bk.On("SetOracleAttestation", ctx, consAddr.String(), ve.OracleAttestations[0].Snapshot, ve.OracleAttestations[0].Attestation).Return(nil)

	res, err := p.PreBlocker(ctx, &req)
	require.NoError(err)
	require.NotNil(res)
}
