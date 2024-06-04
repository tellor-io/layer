package e2e_test

import (
	"encoding/hex"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	collections "cosmossdk.io/collections"
	math "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func (s *E2ETestSuite) TestGovernanceChangesCycleList() {
	require := s.Require()

	govMsgServer := govkeeper.NewMsgServerImpl(s.govKeeper)
	require.NotNil(govMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create bonded validators and reporters
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	valAccAddrs, _, _ := s.CreateValidators(5)
	// repAccAddrs := s.CreateReporters(5, valValAddrs, vals)
	proposer := valAccAddrs[0]
	initCoins := sdk.NewCoin(s.denom, math.NewInt(500*1e6))
	for _, rep := range valAccAddrs {
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, rep, sdk.NewCoins(initCoins)))
	}

	govParams, err := s.govKeeper.Params.Get(s.ctx)
	require.NoError(err)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - submit proposal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	matic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	msgUpdateCycleList := oracletypes.MsgUpdateCyclelist{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Cyclelist: [][]byte{matic},
	}
	anyMsg, err := codectypes.NewAnyWithValue(&msgUpdateCycleList)
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

	proposal, err := govMsgServer.SubmitProposal(s.ctx, &msgSubmitProposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(1))

	proposal1, err := s.govKeeper.Proposals.Get(s.ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusVotingPeriod)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	_, err = s.app.EndBlocker(s.ctx) // end blocker should emit active proposal event
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - vote on proposal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// vote from each validator
	for _, val := range valAccAddrs {
		voteResponse, err := govMsgServer.Vote(s.ctx, &v1.MsgVote{
			ProposalId: proposal.ProposalId,
			Voter:      val.String(),
			Option:     v1.VoteOption(1),
			Metadata:   "vote metadata from validator",
		})
		require.NoError(err)
		require.NotNil(voteResponse)
	}

	// check on vote in collections
	vote, err := s.govKeeper.Votes.Get(s.ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(48 * time.Hour))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - proposal passes
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	// proposal passed
	proposal1, err = s.govKeeper.Proposals.Get(s.ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - check cycle list
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)

	cycleList, err := s.oraclekeeper.GetCyclelist(s.ctx)
	require.NoError(err)
	require.Equal(cycleList, [][]byte{matic})
}
