package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
)

func TestProportionalKeptAmountsRemainder(t *testing.T) {
	cases := []struct {
		name      string
		paid      []int64
		collected int64
		want      []int64
	}{
		{
			name:      "zero collected keeps nothing",
			paid:      []int64{5, 995},
			collected: 0,
			want:      []int64{0, 0},
		},
		{
			name:      "small payer loses the only remainder loya",
			paid:      []int64{5, 995},
			collected: 10,
			want:      []int64{0, 10},
		},
		{
			name:      "three payers split two loya across the tied remainders",
			paid:      []int64{1, 1, 1},
			collected: 2,
			want:      []int64{1, 1, 0},
		},
		{
			name:      "uneven shares sum to the collected total",
			paid:      []int64{1, 2, 3},
			collected: 4,
			want:      []int64{1, 1, 2},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			paid := make([]math.Int, len(tc.paid))
			total := math.ZeroInt()
			for i, amount := range tc.paid {
				paid[i] = math.NewInt(amount)
				total = total.Add(paid[i])
			}
			got := proportionalKeptAmounts(paid, total, math.NewInt(tc.collected))
			sum := math.ZeroInt()
			for i, share := range got {
				require.False(t, share.GT(paid[i]), "kept %s exceeds paid %s", share, paid[i])
				require.Equal(t, math.NewInt(tc.want[i]), share)
				sum = sum.Add(share)
			}
			require.Equal(t, math.NewInt(tc.collected), sum)
		})
	}
}
