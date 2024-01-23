package keeper

import (
	"cosmossdk.io/math"
	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

type ValidatorReportCount struct {
	Power   int64
	Reports int
}

func (k Keeper) AllocateTimeBasedRewards(ctx sdk.Context, reporters []*types.AggregateReporter) {
	totalPower := int64(0)
	reportCounts := make(map[string]ValidatorReportCount)

	for _, r := range reporters {
		reporter, found := reportCounts[r.Reporter]
		if found {
			reporter.Reports++
		} else {
			reporter = ValidatorReportCount{Power: r.Power, Reports: 1}
		}
		reportCounts[r.Reporter] = reporter
		totalPower += r.Power
	}

	tbr := k.getTimeBasedRewards(ctx)
	rewards := make(map[string]sdk.DecCoins)
	for r, c := range reportCounts {
		amount := CalculateRewardAmount(c.Power, int64(c.Reports), totalPower, tbr.Amount)
		rewards[r] = sdk.NewDecCoins(sdk.NewDecCoin(tbr.Denom, amount))
	}
	toDistr := sdk.NewCoins()
	// allocate rewards
	for reporter, reward := range rewards {
		validator, err := k.stakingKeeper.Validator(ctx, sdk.ValAddress(sdk.MustAccAddressFromBech32(reporter)))
		// TODO: return error instead of panic
		if err != nil {
			panic(err)
		}
		k.distrKeeper.AllocateTokensToValidator(ctx, validator, reward)
		coin, _ := reward.TruncateDecimal()
		toDistr = toDistr.Add(coin...)
	}
	if !toDistr.IsZero() {
		// once rewards are allocated, send them to the distribution module
		k.bankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.TimeBasedRewards, distrtypes.ModuleName, toDistr)
	}
}

func (k Keeper) getTimeBasedRewards(ctx sdk.Context) sdk.Coin {
	tbrAccount := k.getTimeBasedRewardsAccount(ctx)
	return k.bankKeeper.GetBalance(ctx, tbrAccount.GetAddress(), types.DefaultBondDenom)
}

func (k Keeper) getTimeBasedRewardsAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, minttypes.TimeBasedRewards)
}

func CalculateRewardAmount(reporterPower, reportsCount, totalPower int64, reward cosmosmath.Int) cosmosmath.Int {
	power := math.LegacyNewDec(reporterPower * reportsCount)
	amount := power.Quo(math.LegacyNewDec(totalPower)).MulTruncate(math.LegacyNewDecFromBigInt(reward.BigInt()))
	return amount.RoundInt()
}
