package keeper

import (
	"context"

	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ReportersReportCount struct {
	Power   uint64
	Reports uint64
	Height  uint64
}

// AllocateRewards distributes rewards to reporters based on their power and number of reports.
// It calculates the reward amount for each reporter and allocates the rewards.
// Finally, it sends the allocated rewards to the apprppopriate module based on the source of the reward.
func (k Keeper) AllocateRewards(ctx context.Context, reporters []*types.AggregateReporter, reward math.Int, fromPool string) error {
	if reward.IsZero() {
		return nil
	}
	// Initialize totalPower to keep track of the total power of all reporters.
	totalPower := uint64(0)
	// reportCounts maps reporter's address to their ValidatorReportCount.
	_reporters := make(map[string]ReportersReportCount)

	// Loop through each reporter to calculate total power and individual report counts.
	for _, r := range reporters {
		reporter, found := _reporters[r.Reporter]
		if found {
			// If the reporter is already in the map, increment their report count.
			reporter.Reports++
		} else {
			// If not found, add the reporter with their initial power and report count set to 1.
			reporter = ReportersReportCount{Power: r.Power, Reports: 1, Height: r.BlockNumber}
		}
		_reporters[r.Reporter] = reporter
		// Add the reporter's power to the total power.
		totalPower += r.Power
	}
	i := len(_reporters)
	totaldist := math.ZeroUint()
	for r, c := range _reporters {
		amount := CalculateRewardAmount(c.Power, c.Reports, totalPower, reward)
		totaldist = totaldist.Add(amount.Value)
		reporterAddr, err := sdk.AccAddressFromBech32(r)
		if err != nil {
			return err
		}
		i--
		if i == 0 {
			amount.Value = amount.Value.Add(math.NewUint(reward.Uint64() * 1e6).Sub(totaldist))
		}
		err = k.AllocateTip(ctx, reporterAddr.Bytes(), amount, c.Height)
		if err != nil {
			return err
		}
	}

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromPool, reportertypes.TipsEscrowPool, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, reward)))
}

func (k Keeper) GetTimeBasedRewards(ctx context.Context) math.Int {
	tbrAccount := k.GetTimeBasedRewardsAccount(ctx)
	balance := k.bankKeeper.GetBalance(ctx, tbrAccount.GetAddress(), layer.BondDenom)
	return balance.Amount
}

func (k Keeper) GetTimeBasedRewardsAccount(ctx context.Context) sdk.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, minttypes.TimeBasedRewards)
}

func CalculateRewardAmount(reporterPower, reportsCount, totalPower uint64, reward math.Int) reportertypes.BigUint {
	normalizedPowerAndReward := math.NewUint((uint64(reporterPower) * uint64(reportsCount) * reward.Uint64()) * 1e6)
	amount := normalizedPowerAndReward.Quo(math.NewUint(totalPower))
	return reportertypes.BigUint{Value: amount}
}

func (k Keeper) AllocateTip(ctx context.Context, addr []byte, amount reportertypes.BigUint, height uint64) error {
	return k.reporterKeeper.DivvyingTips(ctx, addr, amount, height)
}
