package integration_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil"
	utils "github.com/tellor-io/layer/utils"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestAggregateOverMultipleBlocks() {
	// Setup msgServers
	require := s.Require()
	msgServerOracle := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(msgServerOracle)
	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(msgServerReporter)
	msgServerDispute := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	require.NotNil(msgServerDispute)
	msgServerStaking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	require.NotNil(msgServerStaking)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	//---------------------------------------------------------------------------
	// Height 0 - vicky becomes a validator
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	vickyAccAddr := simtestutil.CreateIncrementalAccounts(1)
	vickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(2000*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(vickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, vickyAccAddr[0], sdk.NewCoins(vickyInitCoins)))
	s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, vickyAccAddr[0])

	pubKey := simtestutil.CreateTestPubKeys(1)
	vickyValAddr := simtestutil.ConvertAddrsToValAddrs(vickyAccAddr)
	msgCreateValidator, err := stakingtypes.NewMsgCreateValidator(
		vickyValAddr[0].String(),
		pubKey[0],
		sdk.NewCoin(s.Setup.Denom, math.NewInt(1000*1e6)),
		stakingtypes.Description{Moniker: "created validator"},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(0, 0), math.LegacyNewDecWithPrec(3, 1), math.LegacyNewDecWithPrec(1, 1)),
		math.OneInt(),
	)
	require.NoError(err)

	_, err = msgServerStaking.CreateValidator(s.Setup.Ctx, msgCreateValidator)
	require.NoError(err)

	require.NoError(s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, vickyValAddr[0].String(), []byte("vickyEvmAddr")))

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 1 - Rob delegates to Vicky and selects himself to become a reporter
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// verify vicky is a bonded validator
	vickyValidatorInfo, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, vickyValAddr[0])
	require.NoError(err)
	require.Equal(vickyValidatorInfo.Status, stakingtypes.Bonded)
	require.Equal(vickyValidatorInfo.Tokens, math.NewInt(1000*1e6))

	robPrivKey := secp256k1.GenPrivKey()
	robAccAddr := sdk.AccAddress(robPrivKey.PubKey().Address())
	robInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(robInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, robAccAddr, sdk.NewCoins(robInitCoins)))

	// rob delegates to vicky
	msgDelegate := stakingtypes.NewMsgDelegate(
		robAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(100*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// rob becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   robAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	robReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, robAccAddr)
	require.NoError(err)
	require.Equal(robReporterInfo.Jailed, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - Delwood delegates 250 trb to Vicky
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	delwoodPrivKey := secp256k1.GenPrivKey()
	delwoodAccAddr := sdk.AccAddress(delwoodPrivKey.PubKey().Address())
	delwoodInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(250*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(delwoodInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, delwoodAccAddr, sdk.NewCoins(delwoodInitCoins)))

	msgDelegate = stakingtypes.NewMsgDelegate(
		delwoodAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(250*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - Delwood selects 250 trb to Rob
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(3)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = msgServerReporter.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{
		SelectorAddress: delwoodAccAddr.String(),
		ReporterAddress: robAccAddr.String(),
	})
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - Roman and Ricky become reporters
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(4)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	romanPrivKey := secp256k1.GenPrivKey()
	romanAccAddr := sdk.AccAddress(romanPrivKey.PubKey().Address())
	romanInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(200*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(romanInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, romanAccAddr, sdk.NewCoins(romanInitCoins)))

	// roman delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		romanAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(200*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   romanAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	romanReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, romanAccAddr)
	require.NoError(err)
	require.Equal(romanReporterInfo.Jailed, false)

	rickyPrivKey := secp256k1.GenPrivKey()
	rickyAccAddr := sdk.AccAddress(rickyPrivKey.PubKey().Address())
	rickyInitCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(300*1e6))
	require.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(rickyInitCoins)))
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rickyAccAddr, sdk.NewCoins(rickyInitCoins)))

	// ricky delegates to vicky
	msgDelegate = stakingtypes.NewMsgDelegate(
		rickyAccAddr.String(),
		vickyValAddr[0].String(),
		sdk.NewCoin(s.Setup.Denom, math.NewInt(300*1e6)),
	)
	_, err = msgServerStaking.Delegate(s.Setup.Ctx, msgDelegate)
	require.NoError(err)

	// ricky becomes a reporter
	_, err = msgServerReporter.CreateReporter(s.Setup.Ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   rickyAccAddr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: math.NewInt(1 * 1e6),
	})
	require.NoError(err)
	rickyReporterInfo, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, rickyAccAddr)
	require.NoError(err)
	require.Equal(rickyReporterInfo.Jailed, false)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - only one block left in this cycle list query, pretend empty block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - Rob direct reveals for cycle list at height 6
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(6)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	// Rob direct reveals for cycle list
	currentCycleList, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryId := utils.QueryIDFromData(currentCycleList)
	msgSubmitValue := oracletypes.MsgSubmitValue{
		Creator:   robAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(90_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - Roman and Ricky direct reveal for the same cycle list at height 7
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(7)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	msgSubmitValue = oracletypes.MsgSubmitValue{
		Creator:   romanAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(100_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	msgSubmitValue = oracletypes.MsgSubmitValue{
		Creator:   rickyAccAddr.String(),
		QueryData: currentCycleList,
		Value:     testutil.EncodeValue(110_000),
	}
	_, err = msgServerOracle.SubmitValue(s.Setup.Ctx, &msgSubmitValue)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - Commit window expires, report gets aggregated in endblock
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(8)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Second * 2))
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	aggregate, time, err := s.Setup.Oraclekeeper.GetCurrentAggregateReport(s.Setup.Ctx, queryId)
	require.NoError(err)
	// require.Equal(3, len(aggregate.Reporters))
	// require.Equal(aggregate.AggregateReportIndex, uint64(1))
	require.Equal(aggregate.AggregateValue, testutil.EncodeValue(100_000))
	require.Equal(aggregate.AggregateReporter, romanAccAddr.String())
	require.Equal(aggregate.Height, uint64(7))
	require.Equal(aggregate.AggregatePower, uint64(850))
	require.Equal(aggregate.QueryId, queryId)
	require.Equal(aggregate.MetaId, uint64(2))

	// agg, err := s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, collections.Join(queryId, uint64(time.UnixMilli())))
	// require.NoError(err)
	// require.Equal(3, len(agg.Reporters))

	oracleQuerier := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper)
	microreports, err := oracleQuerier.GetReportsByAggregate(s.Setup.Ctx, &oracletypes.QueryGetReportsByAggregateRequest{
		QueryId:    hex.EncodeToString(queryId),
		Timestamp:  uint64(time.UnixMilli()),
		Pagination: &query.PageRequest{Limit: 100, CountTotal: true},
	})
	require.NoError(err)
	require.Equal(3, len(microreports.MicroReports))
}
