package e2e_test

import (
	"time"

	math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func (s *E2ETestSuite) TestEditingSpec() {
	require := s.Require()

	registryMsgServer := registrykeeper.NewMsgServerImpl(s.registrykeeper)
	require.NotNil(registryMsgServer)
	oracleMsgServer := oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	require.NotNil(oracleMsgServer)
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(reporterMsgServer)

	//---------------------------------------------------------------------------
	// Height 0 - create 1 validator and 1 reporter
	//---------------------------------------------------------------------------
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	valAccAddrs, valValAddrs, vals := s.CreateValidators(1)
	repAccAddrs := s.CreateReporters(1, valValAddrs, vals)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - register a spec for a TWAP query
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	var dataspec = registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         repAccAddrs[0].String(),
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
	}
	_, err = registryMsgServer.RegisterSpec(s.ctx, &registrytypes.MsgRegisterSpec{
		Registrar: repAccAddrs[0].String(),
		QueryType: "TWAP",
		Spec:      dataspec,
	})
	require.NoError(err)

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - tip for eth/usd TWAP
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(2)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

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
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	msgSubmit := oracletypes.MsgSubmitValue{
		Creator:   repAccAddrs[0].String(),
		QueryData: encodedDataSpec,
		Value:     encodeValue(5_000),
	}
	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(7 * time.Second)))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - change spec owner to validator instead of reporter
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(4)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	// claim tip from block 3 submitValue
	msgWithdrawTip := reportertypes.MsgWithdrawTip{
		DelegatorAddress: repAccAddrs[0].String(),
		ValidatorAddress: valValAddrs[0].String(),
	}
	_, err = reporterMsgServer.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)

	var updatedSpec = registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         valAccAddrs[0].String(),
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "asset", FieldType: "string"},
			{Name: "currency", FieldType: "string"},
		},
	}
	msgUpdateSpec := registrytypes.MsgUpdateDataSpec{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		QueryType: "TWAP",
		Spec:      updatedSpec,
	}

	_, err = registryMsgServer.UpdateDataSpec(s.ctx, &msgUpdateSpec)
	require.NoError(err)
	spec, err := s.registrykeeper.GetSpec(s.ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, valAccAddrs[0].String())

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - tip and direct reveal for updated spec
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(5)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	_, err = oracleMsgServer.Tip(s.ctx, &msgTip)
	require.NoError(err)

	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(7 * time.Second)))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - tip, update, then reveal
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(6)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	// claim tip from block 5 submitValue
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

	_, err = registryMsgServer.UpdateDataSpec(s.ctx, &msgUpdateSpec)
	require.NoError(err)
	spec, err = s.registrykeeper.GetSpec(s.ctx, "TWAP")
	require.NoError(err)
	require.Equal(spec.Registrar, repAccAddrs[0].String())

	_, err = oracleMsgServer.SubmitValue(s.ctx, &msgSubmit)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(7 * time.Second)))

	_, err = s.app.EndBlocker(s.ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - claim tip from block 6 submitValue
	//---------------------------------------------------------------------------
	s.ctx = s.ctx.WithBlockHeight(7)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Duration(1 * time.Second)))

	_, err = reporterMsgServer.WithdrawTip(s.ctx, &msgWithdrawTip)
	require.NoError(err)

}
