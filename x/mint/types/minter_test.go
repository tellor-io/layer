package types_test

import (
	"testing"
	time "time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/mint/types"
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
}
