package keeper_test

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const qData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"

var spotSpec = registrytypes.DataSpec{
	DocumentHash:      "",
	ResponseValueType: "uint256",
	AbiComponents: []*registrytypes.ABIComponent{
		{Name: "asset", FieldType: "string"},
		{Name: "currency", FieldType: "string"},
	},
	AggregationMethod: "weighted-median",
	Registrar:         "genesis",
	ReportBlockWindow: 2,
	QueryType:         "spotprice",
}

var bridgeSpec = registrytypes.DataSpec{
	DocumentHash:      "",
	ResponseValueType: "address, string, uint256, uint256",
	AbiComponents: []*registrytypes.ABIComponent{
		{
			Name:            "toLayer",
			FieldType:       "bool",
			NestedComponent: []*registrytypes.ABIComponent{},
		},
		{
			Name:            "depositId",
			FieldType:       "uint256",
			NestedComponent: []*registrytypes.ABIComponent{},
		},
	},
	AggregationMethod: "weighted-mode",
	Registrar:         "genesis",
	ReportBlockWindow: 2000,
	QueryType:         "trbbridge",
}

func (s *KeeperTestSuite) TestSubmitValue() (sdk.AccAddress, []byte) {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.reporterKeeper
	addr := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)
	query := types.QueryMeta{
		Id:                      1,
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               qDataBz,
		QueryType:               "SpotPrice",
		CycleList:               true,
	}

	err = k.Query.Set(s.ctx, collections.Join(queryId, query.Id), query)
	require.NoError(err)
	err = k.QueryDataLimit.Set(s.ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)
	// reporterstake err
	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: qDataBz,
		Value:     value,
	}
	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx := s.ctx.WithBlockHeight(18)

	rk.On("ReporterStake", ctx, addr, queryId).Return(math.NewInt(1_000_000), errors.New("error")).Once()
	_, err = s.msgServer.SubmitValue(ctx, &submitreq)
	require.Error(err)

	// reporterstake less than minStakeAmount
	params, err := k.Params.Get(ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount
	rk.On("ReporterStake", s.ctx, addr, queryId).Return(minStakeAmt.Sub(math.NewInt(100)), nil).Once()
	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	require.Error(err)

	//  good submit
	rk.On("ReporterStake", s.ctx, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(spotSpec, nil).Once()
	res, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	require.NoError(err)
	require.Equal(&types.MsgSubmitValueResponse{Id: 1}, res)

	report, err := s.queryClient.GetReportsbyQid(ctx, &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryId)})
	s.Nil(err)

	microReport := types.MicroReportStrings{
		Reporter:        addr.String(),
		Power:           1,
		QueryType:       "SpotPrice",
		QueryId:         hex.EncodeToString(queryId),
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       uint64(s.ctx.BlockTime().UnixMilli()),
		Cyclelist:       true,
		BlockNumber:     uint64(s.ctx.BlockHeight()),
		MetaId:          query.Id,
	}
	expectedReport := types.QueryMicroReportsResponse{
		MicroReports: []types.MicroReportStrings{microReport},
	}
	require.Equal(expectedReport.MicroReports, report.MicroReports)

	return addr, queryId
}

func (s *KeeperTestSuite) TestSubmitWithNoCreator() {
	// submit value with no creator
	require := s.Require()

	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)

	submitreq := types.MsgSubmitValue{
		QueryData: qDataBz,
		Value:     value,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "invalid creator address")
}

func (s *KeeperTestSuite) TestSubmitWithNoQueryData() {
	// submit value with no query data

	addr := sample.AccAddressBytes()

	submitreq := types.MsgSubmitValue{
		Creator: addr.String(),
		Value:   value,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "query data cannot be empty")
}

func (s *KeeperTestSuite) TestSubmitWithNoValue() {
	// submit value with no value
	require := s.Require()
	addr := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: qDataBz,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "value cannot be empty")
}

func (s *KeeperTestSuite) TestSubmitValueDirectReveal() {
	require := s.Require()
	k := s.oracleKeeper
	repk := s.reporterKeeper
	regk := s.registryKeeper
	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx := s.ctx
	err := k.QueryDataLimit.Set(s.ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)
	regk.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil)
	s.NoError(s.oracleKeeper.RotateQueries(ctx))
	s.NoError(s.oracleKeeper.RotateQueries(ctx))
	s.NoError(s.oracleKeeper.RotateQueries(ctx))
	currentQuery, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	reporter := sample.AccAddressBytes()
	msgSubmitValue := types.MsgSubmitValue{
		Creator:   reporter.String(),
		QueryData: currentQuery,
		Value:     value,
	}

	params, err := k.Params.Get(ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount
	repk.On("ReporterStake", ctx, reporter, utils.QueryIDFromData(currentQuery)).Return(minStakeAmt.Add(math.NewInt(1*1e6)), nil).Once()
	err = k.QueryDataLimit.Set(ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)

	res, err := s.msgServer.SubmitValue(ctx, &msgSubmitValue)
	require.NoError(err)
	require.NotNil(res)

	// check on report
	queryId := utils.QueryIDFromData(currentQuery)
	microReport, err := k.Reports.Get(ctx, collections.Join3(queryId, reporter.Bytes(), uint64(0)))
	require.NoError(err)
	require.NotNil(microReport)
	require.Equal(microReport.AggregateMethod, "weighted-median")
	require.Equal(microReport.BlockNumber, uint64(0))
	require.Equal(microReport.Cyclelist, true)
	require.Equal(microReport.Power, uint64(sdk.TokensToConsensusPower(minStakeAmt.Add(math.NewInt(1*1e6)), layertypes.PowerReduction)))
	require.Equal(microReport.QueryId, queryId)
	require.Equal(microReport.Reporter, reporter.String())
	require.Equal(microReport.Timestamp, ctx.BlockTime())
	require.Equal(microReport.Value, value)
}

func (s *KeeperTestSuite) TestDirectReveal() {
	require := s.Require()
	k := s.oracleKeeper
	regK := s.registryKeeper
	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx := s.ctx
	// returns data spec with report block window set to 3
	regK.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil)
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	// query amount is 0, query expiration is before blocktime, not incycle, should err
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)
	query := types.QueryMeta{
		Id:                      1,
		Amount:                  math.NewInt(0),
		Expiration:              10,
		RegistrySpecBlockWindow: 3,
		QueryData:               qDataBz,
		QueryType:               "SpotPrice",
	}
	reporter := sample.AccAddressBytes()
	votingPower := uint64(sdk.TokensToConsensusPower(math.NewInt(1_000_000), layertypes.PowerReduction))
	isBridgeDeposit := true
	ctx = ctx.WithBlockHeight(15)
	err = k.DirectReveal(ctx, query, qDataBz, value, reporter, votingPower, !isBridgeDeposit)
	require.ErrorContains(err, types.ErrNoTipsNotInCycle.Error())

	// query amount is 0, query expiration + offset is before blocktime, incycle, should set nextId and setValue
	query.Expiration = uint64(ctx.BlockHeight())
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))
	regK.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()
	err = k.DirectReveal(ctx, query, qDataBz, value, reporter, votingPower, isBridgeDeposit)
	require.NoError(err)
	microReport, err := k.Reports.Get(ctx, collections.Join3(queryId, reporter.Bytes(), uint64(1)))
	require.NoError(err)
	require.NotNil(microReport)
	require.Equal(microReport.AggregateMethod, "weighted-median")
	require.Equal(microReport.BlockNumber, uint64(15))
	require.Equal(microReport.Cyclelist, true)
	require.Equal(microReport.QueryId, queryId)
	require.Equal(microReport.Reporter, reporter.String())
	require.Equal(microReport.Timestamp, ctx.BlockTime())
	require.Equal(microReport.Value, value)

	// query amount > 0, query expiration is before blocktime, not in cycle
	query.Amount = math.NewInt(1)
	query.Expiration = uint64(ctx.BlockHeight() - 1)
	query.Id = 4
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))
	regK.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()
	err = k.DirectReveal(ctx, query, qDataBz, value, reporter, votingPower, !isBridgeDeposit)
	require.ErrorContains(err, "submission window expired")

	// query amount > 0, query expiration is before blocktime, in cycle, should setValue
	query.Expiration = 10
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))
	ctx = ctx.WithBlockHeight(15)
	regK.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil).Once()
	err = k.DirectReveal(ctx, query, qDataBz, value, reporter, votingPower, isBridgeDeposit)
	require.NoError(err)
	microReport, err = k.Reports.Get(ctx, collections.Join3(queryId, reporter.Bytes(), uint64(4))) //
	require.NoError(err)
	require.NotNil(microReport)
	require.Equal(microReport.AggregateMethod, "weighted-median")
	require.Equal(microReport.BlockNumber, uint64(15))
	require.Equal(microReport.Cyclelist, true)
	require.Equal(microReport.QueryId, queryId)
	require.Equal(microReport.Reporter, reporter.String())
	require.Equal(microReport.Timestamp, ctx.BlockTime())
	require.Equal(microReport.Value, value)
}

func BenchmarkSubmitValue(b *testing.B) {
	// Setup a new test suite for each benchmark run
	s := new(KeeperTestSuite)
	s.SetupTest()

	// Get the required components
	k := s.oracleKeeper
	rk := s.reporterKeeper
	regk := s.registryKeeper

	// Setup common test data
	addr := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(b, err)
	queryId := utils.QueryIDFromData(qDataBz)

	// Get params for stake amount check
	params, err := k.Params.Get(s.ctx)
	require.NoError(b, err)
	minStakeAmt := params.MinStakeAmount

	// Setup the query
	query := types.QueryMeta{
		Id:                      1,
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               qDataBz,
		QueryType:               "SpotPrice",
		CycleList:               true,
	}

	// Prepare the context
	ctx := s.ctx.WithBlockHeight(18).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	// Setup the submit request
	submitReq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: qDataBz,
		Value:     value,
	}

	b.ResetTimer()
	b.Run("Submit_Value_Success", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create fresh context and setup for each iteration
			iterCtx := ctx.WithBlockHeight(18 + int64(i))

			// Setup fresh state for each iteration
			err = k.Query.Set(iterCtx, collections.Join(queryId, query.Id), query)
			require.NoError(b, err)
			err = k.QueryDataLimit.Set(iterCtx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
			require.NoError(b, err)

			// Setup expectations for each iteration
			rk.On("ReporterStake", iterCtx, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()
			regk.On("GetSpec", iterCtx, "SpotPrice").Return(spotSpec, nil).Once()

			// Run the actual operation we're benchmarking
			_, err := s.msgServer.SubmitValue(iterCtx, &submitReq)
			require.NoError(b, err)
		}
	})
}
