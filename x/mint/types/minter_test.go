package types_test

import (
	stdmath "math"
	"testing"
	time "time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNewMinter(t *testing.T) {
	require := require.New(t)

	minter := types.NewMinter("loya")
	require.NotNil(minter)
	require.Equal(minter.BondDenom, "loya")
}

func TestDefaultMinter(t *testing.T) {
	require := require.New(t)

	minter := types.DefaultMinter()
	require.NotNil(minter)
	require.Equal(minter.BondDenom, "loya")
}

func TestValidate(t *testing.T) {
	require := require.New(t)

	// prevBlockTime is nil
	minter := types.DefaultMinter()
	err := minter.Validate()
	require.ErrorContains(err, "previous block time cannot be nil")

	// prevBlockTime is not nil
	prevBlockTime := time.Unix(1000, 0)
	minter.PreviousBlockTime = &prevBlockTime
	err = minter.Validate()
	require.NoError(err)

	// bondDenom is empty string
	minter.BondDenom = ""
	err = minter.Validate()
	require.ErrorContains(err, "should not be empty string")
}

func TestCalculateBlockProvision(t *testing.T) {
	require := require.New(t)

	// zero time passed
	minter := types.DefaultMinter()
	blockProvision, err := minter.CalculateBlockProvision(time.Unix(1000, 0), time.Unix(1000, 0))
	require.NoError(err)
	require.Equal(blockProvision.Amount, math.ZeroInt())

	// 10 sec passed
	blockProvision, err = minter.CalculateBlockProvision(time.Unix(1010, 0), time.Unix(1000, 0))
	require.NoError(err)
	expectedAmt := math.NewInt(types.DailyMintRate * 10 * 1000 / types.MillisecondsInDay)
	require.Equal(blockProvision.Amount, expectedAmt)

	// curren time before prev time
	blockProvision, err = minter.CalculateBlockProvision(time.Unix(1000, 0), time.Unix(1010, 0))
	require.ErrorContains(err, "cannot be before previous time")
	require.Equal(blockProvision.Amount, sdk.Coin{}.Amount)

	// Gaps past the int64 product limit must mint the real pro-rata amount.
	// The first window wraps negative; the next wraps to a small positive.
	rate := int64(types.DailyMintRate)
	quot := stdmath.MaxInt64 / rate
	for _, elapsed := range []int64{quot + 1, 2 * (quot + 1)} {
		wrapped := math.NewInt(rate * elapsed / types.MillisecondsInDay)
		expected := types.AmountForDailyRate(rate, elapsed)
		require.True(expected.IsPositive())
		require.False(expected.Equal(wrapped))

		current := time.Unix(0, 0).Add(time.Duration(elapsed) * time.Millisecond)
		blockProvision, err = minter.CalculateBlockProvision(current, time.Unix(0, 0))
		require.NoError(err)
		require.Equal(expected, blockProvision.Amount)
	}
}

func TestAmountForDailyRate(t *testing.T) {
	require := require.New(t)

	require.True(types.AmountForDailyRate(0, 1_000).IsZero())
	require.True(types.AmountForDailyRate(-1, types.MillisecondsInDay).IsZero())
	require.True(types.AmountForDailyRate(types.DailyMintRate, 0).IsZero())
	require.True(types.AmountForDailyRate(types.DailyMintRate, -1).IsZero())

	elapsed := int64(10 * 1000)
	require.Equal(
		math.NewInt(types.DailyMintRate*elapsed/types.MillisecondsInDay),
		types.AmountForDailyRate(types.DailyMintRate, elapsed),
	)
}
