package integration_test

import (
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	globalfeetypes "github.com/strangelove-ventures/globalfee/x/globalfee/types"

	collections "cosmossdk.io/collections"
	math "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func (s *IntegrationTestSuite) TestGovernanceChangesGlobalfee() {
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

	valAccAddrs, valAddrs, _ := s.Setup.CreateValidators(5)
	for _, val := range valAddrs {
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

	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - submit proposal
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// fee before proposal
	minfeeparam := s.Setup.GlobalFeekeeper.GetParams(s.Setup.Ctx)
	require.Nil(minfeeparam.MinimumGasPrices)

	msgUpdatefee := globalfeetypes.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Params:    globalfeetypes.Params{MinimumGasPrices: sdk.NewDecCoins(sdk.NewDecCoin(s.Setup.Denom, math.NewInt(1)))},
	}
	anyMsg, err := codectypes.NewAnyWithValue(&msgUpdatefee)
	proposalMsg := []*codectypes.Any{anyMsg}
	require.NoError(err)
	msgSubmitProposal := v1.MsgSubmitProposal{
		Messages:       proposalMsg,
		InitialDeposit: govParams.MinDeposit,
		Proposer:       proposer.String(),
		Title:          "test",
		Summary:        "test",
	}

	proposal, err := govMsgServer.SubmitProposal(s.Setup.Ctx, &msgSubmitProposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(1))

	proposal1, err := s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusVotingPeriod)
	require.Equal(proposal1.Proposer, proposer.String())
	require.Equal(proposal1.Messages, proposalMsg)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx) // end blocker should emit active proposal event
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
	require.Equal(proposal1.Messages, proposalMsg)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - check cycle list
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	minfeeparam = s.Setup.GlobalFeekeeper.GetParams(s.Setup.Ctx)
	require.Equal(minfeeparam.MinimumGasPrices, sdk.NewDecCoins(sdk.NewDecCoin(s.Setup.Denom, math.NewInt(1))))
}
