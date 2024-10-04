package keeper_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestSubmitValue() (sdk.AccAddress, []byte) {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.reporterKeeper
	addr := sample.AccAddressBytes()
	qData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
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
	// reporterstake err
	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: qDataBz,
		Value:     value,
	}
	ctx := s.ctx.WithBlockHeight(18)
	rk.On("ReporterStake", ctx, addr).Return(math.NewInt(1_000_000), errors.New("error")).Once()
	_, err = s.msgServer.SubmitValue(ctx, &submitreq)
	require.Error(err)

	// reporterstake less than minStakeAmount
	params, err := k.Params.Get(ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount
	rk.On("ReporterStake", s.ctx, addr).Return(minStakeAmt.Sub(math.NewInt(100)), nil).Once()
	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	require.Error(err)

	// Submit value transaction with value revealed, this checks if the value is correctly hashed
	rk.On("ReporterStake", s.ctx, addr).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil).Once()

	res, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	require.NoError(err)
	require.Equal(&types.MsgSubmitValueResponse{}, res)

	report, err := s.queryClient.GetReportsbyQid(ctx, &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryId)})
	s.Nil(err)

	microReport := types.MicroReport{
		Reporter:        addr.String(),
		Power:           1,
		QueryType:       "SpotPrice",
		QueryId:         queryId,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     uint64(s.ctx.BlockHeight()),
	}
	expectedReport := types.QueryMicroReportsResponse{
		MicroReports: []types.MicroReport{microReport},
	}
	require.Equal(expectedReport.MicroReports, report.MicroReports)

	return addr, queryId
}

func (s *KeeperTestSuite) TestSubmitWithBadQueryData() {
	// submit value with bad query data
	badQueryData := []byte("invalidQueryData")
	addr := sample.AccAddressBytes()

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: badQueryData,
		Value:     value,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "invalid query data")
}

func (s *KeeperTestSuite) TestSubmitWithBadValue() {
	require := s.Require()
	// submit wrong value but correct salt

	badValue := "00000F4240"
	addr := sample.AccAddressBytes()
	qData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: qDataBz,
		Value:     badValue,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

// func (s *KeeperTestSuite) TestSubmitWithWrongSalt() {
// 	// submit correct value but wrong salt
// 	addr, _, queryData, cid := s.TestCommitValue()

// 	badSalt, err := oracleutils.Salt(32)
// 	s.Nil(err)

// 	submitreq := types.MsgSubmitValue{
// 		Creator:   addr.String(),
// 		QueryData: queryData,
// 		Value:     value,
// 		Salt:      badSalt,
// 		CommitId:  cid,
// 	}
// 	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

// 	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

// 	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
// 	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
// }

// func (s *KeeperTestSuite) TestSubmitAtWrongBlock() {
// 	// try to submit value in same block as commit

// 	addr, salt, queryData, cid := s.TestCommitValue()

// 	submitreq := types.MsgSubmitValue{
// 		Creator:   addr.String(),
// 		QueryData: queryData,
// 		Value:     value,
// 		Salt:      salt,
// 		CommitId:  cid,
// 	}
// 	// Note: No longer relevant since you can reveal early
// 	// _, err := s.msgServer.SubmitValue(s.s.ctx, &submitreq)
// 	// s.ErrorContains(err, "commit reveal window is too early")

// 	// try to submit value 2 blocks after commit
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour))
// 	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
// 	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil) // submitreq.Salt = salt

// 	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
// 	s.ErrorContains(err, "submission window expired")
// }

func (s *KeeperTestSuite) TestSubmitWithNoCreator() {
	// submit value with no creator
	require := s.Require()

	qData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)

	submitreq := types.MsgSubmitValue{
		QueryData: qDataBz,
		Value:     value,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

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

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "query data cannot be empty")
}

func (s *KeeperTestSuite) TestSubmitWithNoValue() {
	// submit value with no value
	require := s.Require()
	addr := sample.AccAddressBytes()
	qData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)

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
	ctx := s.ctx
	regk.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
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
	repk.On("ReporterStake", ctx, reporter).Return(minStakeAmt.Add(math.NewInt(1*1e6)), nil).Once()

	res, err := s.msgServer.SubmitValue(ctx, &msgSubmitValue)
	require.NoError(err)
	require.NotNil(res)
	fmt.Println(res)

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
	ctx := s.ctx
	// returns data spec with report block window set to 3
	regK.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	s.NoError(s.oracleKeeper.RotateQueries(s.ctx))
	// query amount is 0, query expiration is before blocktime, not incycle, should err
	qData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
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
	regK.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil).Once()
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
	regK.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil).Once()
	err = k.DirectReveal(ctx, query, qDataBz, value, reporter, votingPower, !isBridgeDeposit)
	require.ErrorContains(err, "submission window expired")

	// query amount > 0, query expiration is before blocktime, in cycle, should setValue
	query.Expiration = 10
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))
	ctx = ctx.WithBlockHeight(15)
	regK.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil).Once()
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
