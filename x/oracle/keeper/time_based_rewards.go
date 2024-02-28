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
// It calculates the reward amount for each reporter and allocates the rewards to the corresponding validator.
// Finally, it sends the allocated rewards to the distribution module.
func (k Keeper) AllocateTBRRewards(ctx sdk.Context, reporters []*types.AggregateReporter, tip sdk.Coin) error {
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

	// Declare a map to hold the rewards for each reporter.
	rewards := make(map[string]sdk.DecCoins)
	// Calculate rewards for each reporter based on their power and report counts.
	for r, c := range reportCounts {
		amount := CalculateRewardAmount(c.Power, int64(c.Reports), totalPower, tip.Amount)
		rewards[r] = sdk.NewDecCoins(sdk.NewDecCoin(tip.Denom, amount))
	}
	transferAmt := sdk.NewCoins()
	// Allocate calculated rewards to the corresponding reporters.
	for rep, reward := range rewards {
		repoterAddr, err := sdk.AccAddressFromBech32(rep)
		if err != nil {
			return err
		}
		err = k.reporterKeeper.AllocateTokensToReporter(ctx, repoterAddr.Bytes(), reward)
		if err != nil {
			return err
		}
		coin, _ := reward.TruncateDecimal()
		transferAmt = transferAmt.Add(coin...)
	}
	// If there are rewards to distribute, send them to the reporter module.
	if !transferAmt.IsZero() {
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.TimeBasedRewards, reportertypes.ModuleName, transferAmt)
		if err != nil {
			return err
		}
	}
	return nil
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

func (k Keeper) AllocateRewardsToStake(ctx sdk.Context, reporters []*types.AggregateReporter, tip sdk.Coin) error {
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
	// Calculate rewards for each reporter based on their power and report counts.
	for r, c := range reportCounts {
		amount := CalculateRewardAmount(c.Power, int64(c.Reports), totalPower, tip.Amount)
		repAddr, err := sdk.AccAddressFromBech32(r)
		if err != nil {
			return err
		}
		err = k.reporterKeeper.AllocateRewardsToStake(ctx, repAddr, amount)
		if err != nil {
			return err
		}
	}

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(tip))
}
