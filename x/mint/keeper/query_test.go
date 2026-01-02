package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGetExtraRewardsRate(t *testing.T) {
	config.SetupConfig()

	tests := []struct {
		name         string
		setupParams  *types.ExtraRewardParams // nil = no params set
		expectedRate int64
		expectError  bool
	}{
		{
			name:         "default fallback when no params set",
			setupParams:  nil,
			expectedRate: types.DailyMintRate,
			expectError:  false,
		},
		{
			name: "default fallback when rate is zero",
			setupParams: &types.ExtraRewardParams{
				DailyExtraRewards: 0,
				BondDenom:         "loya",
			},
			expectedRate: types.DailyMintRate,
			expectError:  false,
		},
		{
			name: "custom rate",
			setupParams: &types.ExtraRewardParams{
				DailyExtraRewards: 200000000,
				BondDenom:         "loya",
			},
			expectedRate: 200000000,
			expectError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, _, _, ctx := keepertest.MintKeeper(t)
			querier := keeper.NewQuerier(k)

			if tc.setupParams != nil {
				err := k.ExtraRewardParams.Set(ctx, *tc.setupParams)
				require.NoError(t, err)
			}

			resp, err := querier.GetExtraRewardsRate(ctx, &types.QueryGetExtraRewardsRateRequest{})
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedRate, resp.DailyExtraRewards)
		})
	}
}

func TestGetExtraRewardsRate_NilRequest(t *testing.T) {
	config.SetupConfig()
	k, _, _, ctx := keepertest.MintKeeper(t)
	querier := keeper.NewQuerier(k)

	resp, err := querier.GetExtraRewardsRate(ctx, nil)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestGetExtraRewardsPoolBalance(t *testing.T) {
	config.SetupConfig()

	tests := []struct {
		name            string
		poolBalance     math.Int
		expectedBalance sdk.Coin
	}{
		{
			name:            "pool has balance",
			poolBalance:     math.NewInt(1000000),
			expectedBalance: sdk.NewCoin("loya", math.NewInt(1000000)),
		},
		{
			name:            "pool is empty",
			poolBalance:     math.ZeroInt(),
			expectedBalance: sdk.NewCoin("loya", math.ZeroInt()),
		},
		{
			name:            "large balance",
			poolBalance:     math.NewInt(1000000000000),
			expectedBalance: sdk.NewCoin("loya", math.NewInt(1000000000000)),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, ak, bk, ctx := keepertest.MintKeeper(t)
			querier := keeper.NewQuerier(k)

			// Set params
			err := k.ExtraRewardParams.Set(ctx, types.ExtraRewardParams{
				DailyExtraRewards: types.DailyMintRate,
				BondDenom:         "loya",
			})
			require.NoError(t, err)

			// Mock bank balance
			moduleAddr := ak.GetModuleAddress(types.ExtraRewardsPool)
			bk.On("GetBalance", ctx, moduleAddr, "loya").Return(sdk.NewCoin("loya", tc.poolBalance)).Once()

			resp, err := querier.GetExtraRewardsPoolBalance(ctx, &types.QueryGetExtraRewardsPoolBalanceRequest{})
			require.NoError(t, err)
			require.Equal(t, tc.expectedBalance, resp.Balance)
		})
	}
}

func TestGetExtraRewardsPoolBalance_NilRequest(t *testing.T) {
	config.SetupConfig()
	k, _, _, ctx := keepertest.MintKeeper(t)
	querier := keeper.NewQuerier(k)

	resp, err := querier.GetExtraRewardsPoolBalance(ctx, nil)
	require.Error(t, err)
	require.Nil(t, resp)
}
