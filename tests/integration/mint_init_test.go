package integration_test

import (
	"time"

	minttypes "github.com/tellor-io/layer/x/mint/types"

	collections "cosmossdk.io/collections"
	math "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func (s *IntegrationTestSuite) TestGovernanceInitTbr() {
	require := s.Require()
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})

	govMsgServer := govkeeper.NewMsgServerImpl(s.Setup.Govkeeper)
	require.NotNil(govMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create bonded validators and reporters
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	valAccAddrs, valAccountValAddrs, _ := s.Setup.CreateValidators(5)
	for _, val := range valAccountValAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	proposer := valAccAddrs[0]
	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(500*1e6))
	for _, rep := range valAccAddrs {
		s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rep, sdk.NewCoins(initCoins)))
	}

	govParams, err := s.Setup.Govkeeper.Params.Get(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - submit proposal
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// before proposal inflationary rewards are not initialized to minted
	minter, err := s.Setup.Mintkeeper.Minter.Get(s.Setup.Ctx)
	require.NoError(err)
	require.False(minter.Initialized)

	// check tbr should be zero
	tbrAccount := s.Setup.Accountkeeper.GetModuleAccount(s.Setup.Ctx, minttypes.TimeBasedRewards)
	balBeforeInit := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrAccount.GetAddress(), s.Setup.Denom)
	require.True(balBeforeInit.IsZero())
	msgInit := minttypes.MsgInit{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	}
	anyMsg, err := codectypes.NewAnyWithValue(&msgInit)
	proposalMsg := []*codectypes.Any{anyMsg}
	require.NoError(err)
	msgSubmitProposal := v1.MsgSubmitProposal{
		Messages:       proposalMsg,
		InitialDeposit: govParams.MinDeposit,
		Proposer:       proposer.String(),
		Metadata:       "test metadata",
		Title:          "test title",
		Summary:        "test summary",
		Expedited:      false,
	}

	proposal, err := govMsgServer.SubmitProposal(s.Setup.Ctx, &msgSubmitProposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(1))

	proposal1, err := s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - vote on proposal
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// vote from each validator
	for _, val := range valAccAddrs {
		voteResponse, err := govMsgServer.Vote(s.Setup.Ctx, &v1.MsgVote{
			ProposalId: proposal.ProposalId,
			Voter:      val.String(),
			Option:     v1.VoteOption(1),
			Metadata:   "vote metadata from validator",
		})
		require.NoError(err)
		require.NotNil(voteResponse)
	}

	// check on vote in collections
	vote, err := s.Setup.Govkeeper.Votes.Get(s.Setup.Ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(48 * time.Hour))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - proposal passes
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// proposal passed
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, proposer.String())

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - check cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	minter, err = s.Setup.Mintkeeper.Minter.Get(s.Setup.Ctx)
	require.NoError(err)
	require.True(minter.Initialized)
	balAfterInit := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, tbrAccount.GetAddress(), s.Setup.Denom)
	require.True(balAfterInit.Amount.GT(balBeforeInit.Amount))
}
