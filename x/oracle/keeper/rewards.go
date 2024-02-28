package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

type ReportersReportCount struct {
	Power   int64
	Reports int
}

// AllocateRewards distributes rewards to reporters based on their power and number of reports.
// It calculates the reward amount for each reporter and allocates the rewards.
// Finally, it sends the allocated rewards to the appropriate module based on the source of the reward.
func (k Keeper) AllocateRewards(ctx sdk.Context, reporters []*types.AggregateReporter, reward sdk.Coin, toStake bool) error {
	// Initialize totalPower to keep track of the total power of all reporters.
	totalPower := int64(0)
	// reportCounts maps reporter's address to their ValidatorReportCount.
	reportCounts := make(map[string]ReportersReportCount)

	// Loop through each reporter to calculate total power and individual report counts.
	for _, r := range reporters {
		reporter, found := reportCounts[r.Reporter]
		if found {
			// If the reporter is already in the map, increment their report count.
			reporter.Reports++
		} else {
			// If not found, add the reporter with their initial power and report count set to 1.
			reporter = ReportersReportCount{Power: r.Power, Reports: 1}
		}
		reportCounts[r.Reporter] = reporter
		// Add the reporter's power to the total power.
		totalPower += r.Power
	}

	var allocateReward func(ctx sdk.Context, addr []byte, amount math.Int) error
	var from, to string
	if toStake {
		allocateReward = k.AllocateTip
		from = types.ModuleName
		to = stakingtypes.BondedPoolName
	} else {
		allocateReward = k.AllocateTBR
		from = minttypes.TimeBasedRewards
		to = reportertypes.ModuleName

	}
	for r, c := range reportCounts {
		amount := CalculateRewardAmount(c.Power, int64(c.Reports), totalPower, reward.Amount)
		repoterAddr, err := sdk.AccAddressFromBech32(r)
		if err != nil {
			return err
		}
		err = allocateReward(ctx, repoterAddr.Bytes(), amount)
		if err != nil {
			return err
		}
	}

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, from, to, sdk.NewCoins(reward))
}

func (k Keeper) getTimeBasedRewards(ctx sdk.Context) sdk.Coin {
	tbrAccount := k.getTimeBasedRewardsAccount(ctx)
	return k.bankKeeper.GetBalance(ctx, tbrAccount.GetAddress(), types.DefaultBondDenom)
}

func (k Keeper) getTimeBasedRewardsAccount(ctx sdk.Context) sdk.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, minttypes.TimeBasedRewards)
}

func CalculateRewardAmount(reporterPower, reportsCount, totalPower int64, reward math.Int) math.Int {
	power := math.LegacyNewDec(reporterPower * reportsCount)
	amount := power.Quo(math.LegacyNewDec(totalPower)).MulTruncate(math.LegacyNewDecFromBigInt(reward.BigInt()))
	return amount.RoundInt()
}

func (k Keeper) AllocateTBR(ctx sdk.Context, addr []byte, amount math.Int) error {
	reward := sdk.NewDecCoins(sdk.NewDecCoin(types.DefaultBondDenom, amount))
	return k.reporterKeeper.AllocateTokensToReporter(ctx, addr, reward)
}

func (k Keeper) AllocateTip(ctx sdk.Context, addr []byte, amount math.Int) error {
	return k.reporterKeeper.AllocateRewardsToStake(ctx, addr, amount)
}
