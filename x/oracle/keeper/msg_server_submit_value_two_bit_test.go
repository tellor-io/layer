package keeper_test

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app/config"
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oraclemocks "github.com/tellor-io/layer/x/oracle/mocks"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportermocks "github.com/tellor-io/layer/x/reporter/mocks"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// SubmitValue resolves a 2bit agent only when the real reporter keeper reports
// that the signer has no reporter row. This wires that keeper in, instead of
// stubbing ReporterStake and TwoBitReporter.
func TestSubmitValueTwoBitAgentUsesRealReporter(t *testing.T) {
	config.SetupConfig()
	require := require.New(t)

	oracleKey := storetypes.NewKVStoreKey(oracletypes.StoreKey)
	reporterKey := storetypes.NewKVStoreKey(reportertypes.StoreKey)
	db := db.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(oracleKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(reporterKey, storetypes.StoreTypeIAVL, db)
	require.NoError(stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	stakingKeeper := new(reportermocks.StakingKeeper)
	reporterKeeper := reporterkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(reporterKey),
		log.NewNopLogger(),
		authority,
		new(reportermocks.AccountKeeper),
		stakingKeeper,
		new(reportermocks.BankKeeper),
		new(reportermocks.RegistryKeeper),
	)
	commitmentKeeper := new(reportermocks.OracleKeeper)
	commitmentKeeper.On("GetMaxOpenCommitmentForReporter", mock.Anything, mock.Anything).Return(uint64(0), nil).Maybe()
	reporterKeeper.SetOracleKeeper(commitmentKeeper)

	registryKeeper := new(oraclemocks.RegistryKeeper)
	oracleKeeper := oraclekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(oracleKey),
		new(oraclemocks.AccountKeeper),
		new(oraclemocks.BankKeeper),
		registryKeeper,
		reporterKeeper,
		authority,
	)
	oracleKeeper.SetBridgeKeeper(new(oraclemocks.BridgeKeeper))

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	ctx = ctx.WithBlockHeight(10).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	require.NoError(reporterKeeper.Params.Set(ctx, reportertypes.DefaultParams()))
	require.NoError(oracleKeeper.SetParams(ctx, oracletypes.DefaultParams()))
	require.NoError(oracleKeeper.QueryDataLimit.Set(ctx, oracletypes.QueryDataLimit{Limit: oracletypes.InitialQueryDataLimit()}))

	params, err := oracleKeeper.Params.Get(ctx)
	require.NoError(err)
	stake := params.MinStakeAmount

	owner := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	agent := sample.AccAddressBytes()
	require.NoError(reporterKeeper.Reporters.Set(ctx, owner.Bytes(), reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, reportertypes.DefaultMinLoya, "owner")))
	require.NoError(reporterKeeper.Selectors.Set(ctx, selector.Bytes(), reportertypes.NewSelection(owner.Bytes(), 1)))

	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	delegation := stakingtypes.Delegation{
		DelegatorAddress: selector.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           stake.ToLegacyDec(),
	}
	validator := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		Status:          stakingtypes.Bonded,
		Tokens:          stake,
		DelegatorShares: stake.ToLegacyDec(),
	}
	validatorSet := new(reportermocks.ValidatorSet)
	stakingKeeper.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(100), nil)
	stakingKeeper.On("IterateDelegatorDelegations", mock.Anything, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(delegation)
	})
	stakingKeeper.On("GetValidator", mock.Anything, valAddr).Return(validator, nil)

	queryData := mustQueryData(t, oraclekeeper.TwoBitAgenticQueryType)
	queryId := utils.QueryIDFromData(queryData)
	query := oracletypes.QueryMeta{
		Id:         1,
		Amount:     math.NewInt(100_000),
		Expiration: uint64(ctx.BlockHeight()) + 50,
		QueryData:  queryData,
		QueryType:  oraclekeeper.TwoBitAgenticQueryType,
		CycleList:  true,
	}
	require.NoError(oracleKeeper.Query.Set(ctx, collections.Join(queryId, query.Id), query))

	msgServer := oraclekeeper.NewMsgServerImpl(oracleKeeper)
	msg := &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: queryData, Value: value}

	// The agent has no reporter row and no registration, so the real stake lookup
	// returns not-found and submit keeps that error.
	_, err = msgServer.SubmitValue(ctx, msg)
	require.ErrorIs(err, collections.ErrNotFound)

	// A pending request does not authorize reports. The mapping is written on accept.
	require.NoError(reporterKeeper.SetTwoBitReporter(ctx, owner, agent))
	_, err = msgServer.SubmitValue(ctx, msg)
	require.ErrorIs(err, collections.ErrNotFound)

	require.NoError(reporterKeeper.AcceptTwoBitRegistration(ctx, owner, agent))

	// The alias applies only to 2bitAgentic. Another query type keeps the original
	// reporter-not-found error.
	_, err = msgServer.SubmitValue(ctx, &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: mustQueryData(t, "SpotPrice"), Value: value})
	require.ErrorIs(err, collections.ErrNotFound)

	spec := registrytypes.DataSpec{
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		QueryType:         oraclekeeper.TwoBitAgenticQueryType,
	}
	registryKeeper.On("GetSpec", mock.Anything, oraclekeeper.TwoBitAgenticQueryType).Return(spec, nil).Once()

	res, err := msgServer.SubmitValue(ctx, msg)
	require.NoError(err)
	require.Equal(&oracletypes.MsgSubmitValueResponse{Id: query.Id}, res)

	report, err := oracleKeeper.Reports.Get(ctx, collections.Join3(queryId, owner.Bytes(), query.Id))
	require.NoError(err)
	require.Equal(owner.String(), report.Reporter)
	require.Equal(stake.Quo(layertypes.PowerReduction).Uint64(), report.Power)
	require.Equal(oraclekeeper.TwoBitAgenticQueryType, report.QueryType)
	_, err = oracleKeeper.Reports.Get(ctx, collections.Join3(queryId, agent.Bytes(), query.Id))
	require.ErrorIs(err, collections.ErrNotFound)

	// Query type matching ignores case. This is a different query id from the one above.
	foldedData := mustQueryData(t, "2BITAGENTIC")
	foldedID := utils.QueryIDFromData(foldedData)
	folded := query
	folded.Id = 2
	folded.QueryData = foldedData
	require.NoError(oracleKeeper.Query.Set(ctx, collections.Join(foldedID, folded.Id), folded))
	registryKeeper.On("GetSpec", mock.Anything, "2BITAGENTIC").Return(spec, nil).Once()
	res, err = msgServer.SubmitValue(ctx, &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: foldedData, Value: value})
	require.NoError(err)
	require.Equal(&oracletypes.MsgSubmitValueResponse{Id: folded.Id}, res)
	foldedReport, err := oracleKeeper.Reports.Get(ctx, collections.Join3(foldedID, owner.Bytes(), folded.Id))
	require.NoError(err)
	require.Equal(owner.String(), foldedReport.Reporter)

	// Clearing the registration stops the agent. The stored reports stay with the owner.
	require.NoError(reporterKeeper.SetTwoBitReporter(ctx, owner, owner))
	_, err = msgServer.SubmitValue(ctx, &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: foldedData, Value: value})
	require.ErrorIs(err, collections.ErrNotFound)
}
