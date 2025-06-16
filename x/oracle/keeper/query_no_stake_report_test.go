package keeper_test

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func (s *KeeperTestSuite) TestGetReportersNoStakeReports() {
	require := s.Require()

	q := keeper.NewQuerier(s.oracleKeeper)
	require.NotNil(q)

	reporter := sample.AccAddressBytes()

	// no reports
	response, err := q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 0)

	timestamp1 := time.Now().UTC().Add(time.Second * -10)
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp1,
		BlockNumber: 1,
		Value:       "value",
	}
	queryId := []byte("QueryId")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp1.UnixMilli())), report))
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())

	// 2 reports get both
	timestamp2 := time.Now().UTC().Add(time.Second * 10)
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp2,
		BlockNumber: 2,
		Value:       "value2",
	}
	queryId2 := []byte("QueryId2")
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId2, uint64(timestamp2.UnixMilli())), report2))
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 2)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].Reporter, sdk.AccAddress(report2.Reporter).String())

	// return only most recent report
	response, err = q.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 1)
	require.Equal(response.NoStakeReports[0].Value, report2.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report2.Reporter).String())

	fmt.Println("nextKey: ", response.Pagination.NextKey)
	fmt.Println("nextKey string: ", hex.EncodeToString(response.Pagination.NextKey))
}

func (s *KeeperTestSuite) TestGetNoStakeReportsByQueryId() {
	require := s.Require()

	q := keeper.NewQuerier(s.oracleKeeper)
	require.NotNil(q)

	reporter := sample.AccAddressBytes()

	// no reports
	queryId := []byte("QueryId")
	response, err := q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 0)

	timestamp1 := time.Now().UTC().Add(time.Second * -10)
	// 1 report
	report := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp1,
		BlockNumber: 1,
		Value:       "value",
	}
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp1.UnixMilli())), report))
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())

	// 2 reports
	timestamp2 := time.Now().UTC().Add(time.Second * 10)
	s.ctx = s.ctx.WithBlockHeight(2).WithBlockTime(timestamp2)
	report2 := types.NoStakeMicroReport{
		Reporter:    reporter,
		Timestamp:   timestamp2,
		BlockNumber: 2,
		Value:       "value2",
	}
	require.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(s.ctx.BlockTime().UnixMilli())), report2))
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 2)
	require.Equal(response.NoStakeReports[0].Value, report.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp1.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report.Reporter).String())
	require.Equal(response.NoStakeReports[1].Value, report2.Value)
	require.Equal(response.NoStakeReports[1].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[1].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[1].Reporter, sdk.AccAddress(report2.Reporter).String())

	// most recent by query Id (should be timestamp2)
	response, err = q.GetNoStakeReportsByQueryId(s.ctx, &types.QueryGetNoStakeReportsByQueryIdRequest{
		QueryId: hex.EncodeToString(queryId),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	})
	require.NoError(err)
	require.Equal(len(response.NoStakeReports), 1)
	require.Equal(response.NoStakeReports[0].Value, report2.Value)
	require.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp2.UnixMilli()))
	require.Equal(response.NoStakeReports[0].BlockNumber, report2.BlockNumber)
	require.Equal(response.NoStakeReports[0].Reporter, sdk.AccAddress(report2.Reporter).String())

	fmt.Println("nextKey: ", response.Pagination.NextKey)
	fmt.Println("nextKey string: ", hex.EncodeToString(response.Pagination.NextKey))
}

func (s *KeeperTestSuite) TestGetReportersNoStakeReportsMixQueryIds() {
	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	trbBridqeMixed := []string{
		`["true","1"]`,
		`["true","2"]`,
		`["true","3"]`,
		`["true","4"]`,
		`["true","5"]`,
		`["true", "6"]`,
		`["true", "7"]`,
		`["true", "8"]`,
		`["true", "9"]`,
	}
	reporter := sample.AccAddressBytes()
	var timestamp time.Time
	for i, v := range trbBridqeMixed {
		querydata, err := spec.EncodeData("TRBBridge", v)
		s.NoError(err)
		queryId := crypto.Keccak256(querydata)
		timestamp = time.Now().UTC().Add(time.Second * time.Duration(i))
		report := types.NoStakeMicroReport{
			Reporter:    reporter,
			Timestamp:   timestamp,
			BlockNumber: uint64(i + 1),
			Value:       fmt.Sprintf("value%d", i+1),
		}
		fmt.Println(report.BlockNumber)
		s.NoError(s.oracleKeeper.NoStakeReports.Set(s.ctx, collections.Join(queryId, uint64(timestamp.UnixMilli())), report))
	}
	response, err := s.queryClient.GetReportersNoStakeReports(s.ctx, &types.QueryGetReportersNoStakeReportsRequest{
		Reporter: reporter.String(),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	})
	s.NoError(err)
	// should be the last block number ie 10
	s.Equal(response.NoStakeReports[0].BlockNumber, uint64(9))
	s.Equal(response.NoStakeReports[0].Timestamp, uint64(timestamp.UnixMilli()))
}
