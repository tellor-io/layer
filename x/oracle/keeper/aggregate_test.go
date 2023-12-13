package keeper_test

import (
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestSetAggregatedReport() {
	require := s.Require()
	ctx := s.ctx
	// Create a new instance of the keeper
	k := s.oracleKeeper
	reportsStore := k.ReportsStore(ctx)
	fmt.Println("reportsStore: ", reportsStore)

	// Set up test data
	value1 := "a"  //10
	value2 := "14" //20
	value3 := "1e" //30
	fmt.Println("Addr.String(): ", Addr.String())
	reports := types.Reports{
		MicroReports: []*types.MicroReport{
			{
				Reporter:        Addr.String(),
				Power:           1000000000000,
				QueryType:       "SpotPrice",
				QueryId:         "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
				AggregateMethod: "weighted-median",
				Value:           value1,
				BlockNumber:     s.ctx.BlockHeight(),
				Timestamp:       s.ctx.BlockTime(),
			},
			{
				Reporter:        Addr.String(),
				Power:           1000000000000,
				QueryType:       "SpotPrice",
				QueryId:         "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
				AggregateMethod: "weighted-median",
				Value:           value2,
				BlockNumber:     s.ctx.BlockHeight(),
				Timestamp:       s.ctx.BlockTime(),
			},
			{
				Reporter:        Addr.String(),
				Power:           1000000000000,
				QueryType:       "SpotPrice",
				QueryId:         "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
				AggregateMethod: "weighted-median",
				Value:           value3,
				BlockNumber:     s.ctx.BlockHeight(),
				Timestamp:       s.ctx.BlockTime(),
			},
		},
	}

	fmt.Println("Addr.String(): ", Addr.String())

	// Marshal and store the test data
	// Call the `Marshal` function to serialize the data
	bz, err := types.ModuleCdc.Marshal(&reports)
	require.Nil(err)

	// set ReportsStore with the serialized data
	k.ReportsStore(s.ctx).Set(types.BlockKey(s.ctx.BlockHeight()), bz)

	// Call the function under test
	k.SetAggregatedReport(s.ctx)

	// check that reports are aggregated
	var aggregatedReport types.Aggregate
	store := k.AggregateStore(s.ctx)
	bz = store.Get([]byte(fmt.Sprintf("%s-%d", aggregatedReport.QueryId, s.ctx.BlockHeight())))
	types.ModuleCdc.MustUnmarshal(bz, &aggregatedReport)
	response := &types.QueryGetAggregatedReportResponse{Report: &aggregatedReport}
	fmt.Println("response: ", response)
	// TODO: Add assertions for the expected behavior
	require.Equal(s.T(), value1, aggregatedReport.AggregateValue)
}
