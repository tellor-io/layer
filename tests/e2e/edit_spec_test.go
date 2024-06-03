package e2e_test

import (
	"time"

	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	collections "cosmossdk.io/collections"
	math "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func (s *E2ETestSuite) TestEditSpec() {
	require := s.Require()

	registryMsgServer := registrykeeper.NewMsgServerImpl(s.registrykeeper)
	require.NotNil(registryMsgServer)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(oracleMsgServer)
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(reporterMsgServer)
	govMsgServer := govkeeper.NewMsgServerImpl(s.govKeeper)
	require.NotNil(govMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create 1 validator and 1 reporter
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	valAccAddrs, valValAddrs, vals := s.CreateValidators(1)
	repAccAddrs := s.CreateReporters(1, valValAddrs, vals)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - register a spec for a TWAP query, registrar is reporter
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	abiComponents := []*registrytypes.ABIComponent{
		{Name: "asset", FieldType: "string"},
		{Name: "currency", FieldType: "string"},
	}
	dataspec := registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         repAccAddrs[0].String(),
		AbiComponents:     abiComponents,
	}
	_, err = registryMsgServer.RegisterSpec(s.ctx, &registrytypes.MsgRegisterSpec{
		Registrar: repAccAddrs[0].String(),
		QueryType: "TWAP",
		Spec:      dataspec,
	})
	require.NoError(err)
	spec, err := s.registrykeeper.GetSpec(s.ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, repAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - tip for eth/usd TWAP
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(2)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	encodedDataSpec, err := dataspec.EncodeData("TWAP", `["eth","usd"]`)
	require.NoError(err)

	// mint coins to val so they can tip
	initCoins := sdk.NewCoin(s.denom, math.NewInt(100*1e6))
	s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, valAccAddrs[0], sdk.NewCoins(initCoins)))

	msgTip := oracletypes.MsgTip{
		Tipper:    valAccAddrs[0].String(),
		QueryData: encodedDataSpec,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(1*1e6)),
	}
	tipResponse, err := oracleMsgServer.Tip(s.ctx, &msgTip)
	require.NoError(err)
	require.NotNil(tipResponse)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - direct reveal for eth/usd TWAP
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(3)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	msgSubmit := oracletypes.MsgSubmitValue{
		Creator:   repAccAddrs[0].String(),
		QueryData: encodedDataSpec,
		Value:     encodeValue(5_000),
	}
	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - submit and vote on proposal for spec owner to be validator instead of reporter
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(4)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	msgWithdrawTip := reportertypes.MsgWithdrawTip{
		DelegatorAddress: repAccAddrs[0].String(),
		ValidatorAddress: valValAddrs[0].String(),
	}
	_, err = reporterMsgServer.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)

	govParams, err := s.govKeeper.Params.Get(s.ctx)
	require.NoError(err)
	updatedSpec := registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         valAccAddrs[0].String(),
		AbiComponents:     abiComponents,
	}
	msgUpdateSpec := registrytypes.MsgUpdateDataSpec{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		QueryType: "TWAP",
		Spec:      updatedSpec,
	}

	anyMsg, err := codectypes.NewAnyWithValue(&msgUpdateSpec)
	proposalMsg := []*codectypes.Any{anyMsg}
	require.NoError(err)
	msgSubmitProposal := v1.MsgSubmitProposal{
		Messages:       proposalMsg,
		InitialDeposit: govParams.MinDeposit,
		Proposer:       valAccAddrs[0].String(),
		Metadata:       "test metadata",
		Title:          "test title",
		Summary:        "test summary",
		Expedited:      false,
	}

	proposal, err := govMsgServer.SubmitProposal(s.ctx, &msgSubmitProposal)
	require.NoError(err)
	spec, err = s.registrykeeper.GetSpec(s.ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, repAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	voteResponse, err := govMsgServer.Vote(s.ctx, &v1.MsgVote{
		ProposalId: proposal.ProposalId,
		Voter:      valAccAddrs[0].String(),
		Option:     v1.VoteOption(1),
		Metadata:   "vote metadata from validator",
	})
	require.NoError(err)
	require.NotNil(voteResponse)

	vote, err := s.govKeeper.Votes.Get(s.ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(48 * time.Hour))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - vote passes, tip and direct reveal for updated spec
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(5)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	// proposal passed
	proposal1, err := s.govKeeper.Proposals.Get(s.ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, valAccAddrs[0].String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	spec, err = s.registrykeeper.GetSpec(s.ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, valAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	_, err = oracleMsgServer.Tip(s.ctx, &msgTip)
	require.NoError(err)

	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - tip, submit proposal to change registar to reporter, then reveal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(6)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	_, err = reporterMsgServer.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)

	_, err = oracleMsgServer.Tip(s.ctx, &msgTip)
	require.NoError(err)

	updatedSpec = registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         repAccAddrs[0].String(),
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
	}
	msgUpdateSpec = registrytypes.MsgUpdateDataSpec{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		QueryType: "TWAP",
		Spec:      updatedSpec,
	}
	anyMsg, err = codectypes.NewAnyWithValue(&msgUpdateSpec)
	proposalMsg = []*codectypes.Any{anyMsg}
	require.NoError(err)
	msgSubmitProposal = v1.MsgSubmitProposal{
		Messages:       proposalMsg,
		InitialDeposit: govParams.MinDeposit,
		Proposer:       valAccAddrs[0].String(),
		Metadata:       "test metadata",
		Title:          "test title",
		Summary:        "test summary",
		Expedited:      false,
	}

	proposal, err = govMsgServer.SubmitProposal(s.ctx, &msgSubmitProposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(2))

	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - vote on proposal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(7)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	_, err = reporterMsgServer.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)

	voteResponse, err = govMsgServer.Vote(s.ctx, &v1.MsgVote{
		ProposalId: proposal.ProposalId,
		Voter:      valAccAddrs[0].String(),
		Option:     v1.VoteOption(1),
		Metadata:   "vote metadata from validator",
	})
	require.NoError(err)
	require.NotNil(voteResponse)

	vote, err = s.govKeeper.Votes.Get(s.ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(48 * time.Hour))
	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - proposal passes
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(8)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second))

	// proposal passed
	proposal1, err = s.govKeeper.Proposals.Get(s.ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, valAccAddrs[0].String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	spec, err = s.registrykeeper.GetSpec(s.ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, repAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	_, err = oracleMsgServer.Tip(s.ctx, &msgTip)
	require.NoError(err)

	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(7 * time.Second))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)
}
