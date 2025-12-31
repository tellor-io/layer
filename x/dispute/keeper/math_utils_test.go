package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/keeper"

	"cosmossdk.io/math"
)

func TestCalculateRefundAmount(t *testing.T) {
	tests := []struct {
		name            string
		payerFee        math.Int
		totalFeeRd1     math.Int
		disputeFeeTotal math.Int
		expectedAmt     math.Int
		expectedDust    math.Int
	}{
		{
			name:            "full refund, single payer",
			payerFee:        math.NewInt(1000),
			totalFeeRd1:     math.NewInt(1000),
			disputeFeeTotal: math.NewInt(1000),
			// Pot = 1000 - 50 = 950
			// Share = 1.0 * 950 = 950
			expectedAmt:  math.NewInt(950),
			expectedDust: math.ZeroInt(),
		},
		{
			name:            "2 even payers",
			payerFee:        math.NewInt(500),
			totalFeeRd1:     math.NewInt(1000),
			disputeFeeTotal: math.NewInt(1000),
			// 95% of (500/1000) of 1000 = 475
			expectedAmt:  math.NewInt(475),
			expectedDust: math.ZeroInt(),
		},
		{
			name:            "zero fee",
			payerFee:        math.ZeroInt(),
			totalFeeRd1:     math.NewInt(100),
			disputeFeeTotal: math.NewInt(100),
			expectedAmt:     math.ZeroInt(),
			expectedDust:    math.ZeroInt(),
		},
		{
			name:            "large numbers",
			payerFee:        math.NewInt(123456789000000), // 123,456,789 trb
			totalFeeRd1:     math.NewInt(987654321000000), // 987,654,321 trb
			disputeFeeTotal: math.NewInt(987654321000000), // 987,654,321 trb
			// 95% of 12.5% of 987654321000000 = 117283949550000
			expectedAmt:  math.NewInt(117283949550000),
			expectedDust: math.ZeroInt(),
		},
		{
			name:            "dust",
			payerFee:        math.NewInt(333),
			totalFeeRd1:     math.NewInt(1000),
			disputeFeeTotal: math.NewInt(1000),
			// 95% of (333/1000) of 1000 = 316.35
			expectedAmt:  math.NewInt(316),
			expectedDust: math.NewInt(350000), // 0.35 loya * 1e6
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println(tc.name)
			amt, dust := keeper.CalculateRefundAmount(tc.payerFee, tc.totalFeeRd1, tc.disputeFeeTotal)
			require.True(t, tc.expectedAmt.Equal(amt), "expected amount %s, got %s", tc.expectedAmt, amt)
			require.True(t, tc.expectedDust.Equal(dust), "expected dust %s, got %s", tc.expectedDust, dust)
		})
	}
}

func TestCalculateReporterBondRewardAmount(t *testing.T) {
	tests := []struct {
		name          string
		payerFee      math.Int
		totalFeesPaid math.Int
		reporterBond  math.Int
		expectedAmt   math.Int
		expectedDust  math.Int
	}{
		{
			name:          "full reward, single payer",
			payerFee:      math.NewInt(100),
			totalFeesPaid: math.NewInt(100),
			reporterBond:  math.NewInt(1000),
			// 100% of 1000 loya = 1000 loya
			expectedAmt:  math.NewInt(1000),
			expectedDust: math.ZeroInt(),
		},
		{
			name:          "split reward, 2 payers",
			payerFee:      math.NewInt(100),
			totalFeesPaid: math.NewInt(200),
			reporterBond:  math.NewInt(1000),
			// 1/2 of 1000 loya = 500 loya
			expectedAmt:  math.NewInt(500),
			expectedDust: math.ZeroInt(),
		},
		{
			name:          "dust scenario",
			payerFee:      math.NewInt(100),
			totalFeesPaid: math.NewInt(300),
			reporterBond:  math.NewInt(1000),
			// 1/3 of 1000 loya = 333.333333... loya
			expectedAmt:  math.NewInt(333),
			expectedDust: math.NewInt(333333),
		},
		{
			name:          "large numbers",
			payerFee:      math.NewInt(123456789000000),  // 123,456,789 trb
			totalFeesPaid: math.NewInt(987654321000000),  // 987,654,321 trb
			reporterBond:  math.NewInt(1000000000000000), // 1,000,000,000 trb
			// (123456789 trb / 987654321 trb) * 1,000,000,000 trb = 124999998.860937500014238281249822021484 trb
			expectedAmt:  math.NewInt(124999998860937),
			expectedDust: math.NewInt(500014), // 500014238281249822021484 cut down to 6 decimal places
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			amt, dust := keeper.CalculateReporterBondRewardAmount(tc.payerFee, tc.totalFeesPaid, tc.reporterBond)
			require.True(t, tc.expectedAmt.Equal(amt), "expected amount %s, got %s", tc.expectedAmt, amt)
			require.True(t, tc.expectedDust.Equal(dust), "expected dust %s, got %s", tc.expectedDust, dust)
		})
	}
}
