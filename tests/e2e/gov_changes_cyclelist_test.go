package e2e_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil"
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

	govMsgServer := govkeeper.NewMsgServerImpl(s.Setup.Govkeeper)
	require.NotNil(govMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create bonded validators and reporters
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	valAccAddrs, _, _ := s.Setup.CreateValidators(5)

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
	s.Setup.Ctx = testutil.WithBlockTime(s.Setup.Ctx, s.Setup.Ctx.HeaderInfo().Time.Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
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

	proposal, err := govMsgServer.SubmitProposal(s.Setup.Ctx, &msgSubmitProposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(1))

	proposal1, err := s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusVotingPeriod)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx) // end blocker should emit active proposal event
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - vote on proposal
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	s.Setup.Ctx = testutil.WithBlockTime(s.Setup.Ctx, s.Setup.Ctx.HeaderInfo().Time.Add(time.Second))
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
	s.Setup.Ctx = testutil.WithBlockTime(s.Setup.Ctx, s.Setup.Ctx.HeaderInfo().Time.Add(48*time.Hour))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - proposal passes
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	s.Setup.Ctx = testutil.WithBlockTime(s.Setup.Ctx, s.Setup.Ctx.HeaderInfo().Time.Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// proposal passed
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - check cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	s.Setup.Ctx = testutil.WithBlockTime(s.Setup.Ctx, s.Setup.Ctx.HeaderInfo().Time.Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	cycleList, err := s.Setup.Oraclekeeper.GetCyclelist(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(cycleList, [][]byte{matic})
}
