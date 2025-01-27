package e2e_test

import (
	"time"

	"github.com/tellor-io/layer/testutil"
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

	registryMsgServer := registrykeeper.NewMsgServerImpl(s.Setup.Registrykeeper)
	require.NotNil(registryMsgServer)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(oracleMsgServer)
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(reporterMsgServer)
	govMsgServer := govkeeper.NewMsgServerImpl(s.Setup.Govkeeper)
	require.NotNil(govMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create 1 validator and 1 reporter
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	valAccAddrs, valValAddrs, _ := s.Setup.CreateValidators(1)
	// valAccAddrs := s.CreateReporters(1, valValAddrs, vals)
	for _, val := range valValAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	_, err = reporterMsgServer.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{ReporterAddress: valAccAddrs[0].String(), CommissionRate: reportertypes.DefaultMinCommissionRate, MinTokensRequired: reportertypes.DefaultMinTrb})
	require.NoError(err)
	//---------------------------------------------------------------------------
	// Height 1 - register a spec for a TWAP query, registrar is reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	abiComponents := []*registrytypes.ABIComponent{
		{Name: "asset", FieldType: "string"},
		{Name: "currency", FieldType: "string"},
	}
	dataspec := registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         valAccAddrs[0].String(),
		AbiComponents:     abiComponents,
		ReportBlockWindow: 1,
	}
	_, err = registryMsgServer.RegisterSpec(s.Setup.Ctx, &registrytypes.MsgRegisterSpec{
		Registrar: valAccAddrs[0].String(),
		QueryType: "TWAP",
		Spec:      dataspec,
	})
	require.NoError(err)
	spec, err := s.Setup.Registrykeeper.GetSpec(s.Setup.Ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, valAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - tip for eth/usd TWAP
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	encodedDataSpec, err := dataspec.EncodeData("TWAP", `["eth","usd"]`)
	require.NoError(err)

	// mint coins to val so they can tip
	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6))
	s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, valAccAddrs[0], sdk.NewCoins(initCoins)))

	msgTip := oracletypes.MsgTip{
		Tipper:    valAccAddrs[0].String(),
		QueryData: encodedDataSpec,
		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1*1e6)),
	}
	tipResponse, err := oracleMsgServer.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)
	require.NotNil(tipResponse)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - direct reveal for eth/usd TWAP
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(3)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	msgSubmit := oracletypes.MsgSubmitValue{
		Creator:   valAccAddrs[0].String(),
		QueryData: encodedDataSpec,
		Value:     testutil.EncodeValue(5_000),
	}
	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &msgSubmit)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - submit and vote on proposal for spec owner to be validator instead of reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(4)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	msgWithdrawTip := reportertypes.MsgWithdrawTip{
		SelectorAddress:  valAccAddrs[0].String(),
		ValidatorAddress: valValAddrs[0].String(),
	}
	_, err = reporterMsgServer.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
	require.NoError(err)

	govParams, err := s.Setup.Govkeeper.Params.Get(s.Setup.Ctx)
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

	proposal, err := govMsgServer.SubmitProposal(s.Setup.Ctx, &msgSubmitProposal)
	require.NoError(err)
	spec, err = s.Setup.Registrykeeper.GetSpec(s.Setup.Ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, valAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	voteResponse, err := govMsgServer.Vote(s.Setup.Ctx, &v1.MsgVote{
		ProposalId: proposal.ProposalId,
		Voter:      valAccAddrs[0].String(),
		Option:     v1.VoteOption(1),
		Metadata:   "vote metadata from validator",
	})
	require.NoError(err)
	require.NotNil(voteResponse)

	vote, err := s.Setup.Govkeeper.Votes.Get(s.Setup.Ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(48 * time.Hour))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - vote passes, tip and direct reveal for updated spec
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	// proposal passed
	proposal1, err := s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, valAccAddrs[0].String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	spec, err = s.Setup.Registrykeeper.GetSpec(s.Setup.Ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, valAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	_, err = oracleMsgServer.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)

	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &msgSubmit)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - tip, submit proposal to change registar to reporter, then reveal
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(6)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	_, err = reporterMsgServer.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
	require.NoError(err)

	_, err = oracleMsgServer.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)

	updatedSpec = registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         valAccAddrs[0].String(),
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

	proposal, err = govMsgServer.SubmitProposal(s.Setup.Ctx, &msgSubmitProposal)
	require.NoError(err)
	require.Equal(proposal.ProposalId, uint64(2))

	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &msgSubmit)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - vote on proposal
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(7)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	_, err = reporterMsgServer.WithdrawTip(s.Setup.Ctx, &msgWithdrawTip)
	require.NoError(err)

	voteResponse, err = govMsgServer.Vote(s.Setup.Ctx, &v1.MsgVote{
		ProposalId: proposal.ProposalId,
		Voter:      valAccAddrs[0].String(),
		Option:     v1.VoteOption(1),
		Metadata:   "vote metadata from validator",
	})
	require.NoError(err)
	require.NotNil(voteResponse)

	vote, err = s.Setup.Govkeeper.Votes.Get(s.Setup.Ctx, collections.Join(proposal.ProposalId, valAccAddrs[0]))
	require.NoError(err)
	require.Equal(vote.ProposalId, proposal.ProposalId)
	require.Equal(vote.Voter, valAccAddrs[0].String())
	require.Equal(vote.Metadata, "vote metadata from validator")

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(48 * time.Hour))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - proposal passes
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(8)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second))

	// proposal passed
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal.ProposalId)
	require.NoError(err)
	require.Equal(proposal1.Status, v1.StatusPassed)
	require.Equal(proposal1.Proposer, valAccAddrs[0].String())
	require.Equal(proposal1.TotalDeposit, govParams.MinDeposit)
	require.Equal(proposal1.Messages, proposalMsg)
	require.Equal(proposal1.Metadata, "test metadata")
	require.Equal(proposal1.Title, "test title")
	require.Equal(proposal1.Summary, "test summary")
	require.Equal(proposal1.Expedited, false)

	spec, err = s.Setup.Registrykeeper.GetSpec(s.Setup.Ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, valAccAddrs[0].String())
	require.Equal(spec.AbiComponents, abiComponents)
	require.Equal(spec.ResponseValueType, "uint256")
	require.Equal(spec.AggregationMethod, "weighted-median")

	_, err = oracleMsgServer.Tip(s.Setup.Ctx, &msgTip)
	require.NoError(err)

	_, err = oracleMsgServer.SubmitValue(s.Setup.Ctx, &msgSubmit)
	require.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(7 * time.Second))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)
}
