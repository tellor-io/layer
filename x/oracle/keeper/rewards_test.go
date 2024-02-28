package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/oracle/keeper"
)

var reward = math.NewInt(100)

func TestCalculateRewardAmount(t *testing.T) {
	testCases := []struct {
		name           string
		reporter       []keeper.ReportersReportCount
		reporterPowers []int64
		totalPower     int64
		reportsCount   int64
		expectedAmount []math.Int
	}{
		{
			name:           "Test all reporters report",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 1}, {Power: 30, Reports: 1}, {Power: 40, Reports: 1}},
			expectedAmount: []math.Int{math.NewInt(10), math.NewInt(20), math.NewInt(30), math.NewInt(40)},
			totalPower:     100, // 40 + 30 + 20 + 10

		},
		{
			name:           "only 1 reports",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 0}, {Power: 30, Reports: 0}, {Power: 40, Reports: 0}},
			expectedAmount: []math.Int{math.NewInt(100), math.NewInt(0), math.NewInt(0), math.NewInt(0)},
			totalPower:     10,
		},
		{
			name:           "only 1 and 3 reports one report, a single queryId",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 1}, {Power: 20, Reports: 0}, {Power: 30, Reports: 1}, {Power: 40, Reports: 0}},
			expectedAmount: []math.Int{math.NewInt(25), math.NewInt(0), math.NewInt(75), math.NewInt(0)},
			totalPower:     40, // 30 + 10
		},
		{
			name:           "all reporters report, a two queryIds",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 2}, {Power: 30, Reports: 2}, {Power: 40, Reports: 2}},
			expectedAmount: []math.Int{math.NewInt(10), math.NewInt(20), math.NewInt(30), math.NewInt(40)},
			totalPower:     200,
		},
		{
			name:           "all reporters report single, and 1 reports two queryIds",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 1}, {Power: 30, Reports: 1}, {Power: 40, Reports: 1}},
			expectedAmount: []math.Int{math.NewInt(18), math.NewInt(18), math.NewInt(27), math.NewInt(36)},
			totalPower:     110, // 40 + 30 + 20 + (10 * 2)
		},
		{
			name:           "all reporters report single, 1 and 3 report a second queryId",
			reporter:       []keeper.ReportersReportCount{{Power: 10, Reports: 2}, {Power: 20, Reports: 1}, {Power: 30, Reports: 2}, {Power: 40, Reports: 1}},
			expectedAmount: []math.Int{math.NewInt(14), math.NewInt(14), math.NewInt(43), math.NewInt(29)},
			totalPower:     140, // 40 + (30 * 2) + 20 + (10 * 2)
		},
	}
	for _, tc := range testCases {
		expectedTotalReward := math.ZeroInt()
		t.Run(tc.name, func(t *testing.T) {
			for i, r := range tc.reporter {
				amount := keeper.CalculateRewardAmount(r.Power, int64(r.Reports), tc.totalPower, reward)
				require.Equal(t, amount, tc.expectedAmount[i])
				expectedTotalReward = expectedTotalReward.Add(amount)
			}
		})
		tolerance := reward.Sub(math.OneInt()) // when rounded
		withinTolerance := expectedTotalReward.Equal(reward) || expectedTotalReward.Equal(tolerance)
		require.True(t, withinTolerance, "reward amount should be within tolerance")
	}
}
