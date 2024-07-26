package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
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

	sigA, sigB, _ := testutils.GenerateSignatures(s.T())
	valAddr := sample.AccAddressBytes()
	_ = sdk.ConsAddress(valAddr)
	voteExt := app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{
				Attestation: []byte("attestation"),
			},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: sigA,
			SignatureB: sigB,
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: []byte("signature"),
			Timestamp: uint64(s.ctx.BlockTime().Unix()),
		},
	}
	_, err := json.Marshal(voteExt)
	require.NoError(err)
}

// func (s *ProposalHandlerTestSuite) TestValidatorOperatorAddressFromVote_NotFound() {
// 	require := s.Require()
// 	p := s.proposalHandler
// 	require.NotNil(p)
// 	sk := s.stakingKeeper
// 	require.NotNil(sk)
// 	ctx := s.ctx
// 	require.NotNil(ctx)
// }

func (s *ProposalHandlerTestSuite) TestCheckInitialSignaturesFromLastCommit() {
	require := s.Require()
	p := s.proposalHandler
	bk := s.bridgeKeeper
	require.NotNil(bk)
	ctx := s.ctx
	require.NotNil(ctx)

	// not BlockIDFlagCommit vote
	valAccAddr1 := sample.AccAddressBytes()
	val1 := abcitypes.Validator{
		Address: valAccAddr1,
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
	sigA, sigB, addressExpected := testutils.GenerateSignatures(s.T())
	fmt.Println("addressExpected: ", addressExpected, "\nsigA: ", sigA, "\nsigB: ", sigB)

	voteExt := app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{
				Attestation: []byte("attestation"),
			},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: sigA,
			SignatureB: sigB,
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: []byte("signature"),
			Timestamp: uint64(s.ctx.BlockTime().Unix()),
		},
	}
	voteExtBytes, err := json.Marshal(voteExt)
	require.NoError(err)

	ext.Votes = []abcitypes.ExtendedVoteInfo{
		{
			Validator:     val1,
			BlockIdFlag:   cmtproto.BlockIDFlagCommit,
			VoteExtension: voteExtBytes,
		},
	}

	bk.On("EVMAddressFromSignatures", ctx, voteExt.InitialSignature.SignatureA, voteExt.InitialSignature.SignatureB).Return(addressExpected, nil)
	res1, res2, err = p.CheckInitialSignaturesFromLastCommit(ctx, ext)
	require.NoError(err)
	require.Equal(val1.Address, res1)
	require.Equal(val1.Address, res2)
}
