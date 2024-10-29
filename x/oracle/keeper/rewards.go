package keeper

import (
	"context"
	"sort"

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

	// Use a struct to hold reporter info
	type ReporterInfo struct {
		address string
		data    ReportersReportCount
	}

	// First pass: collect data in map
	reportersMap := make(map[string]ReportersReportCount)
	for _, r := range reporters {
		reporter, found := reportersMap[r.Reporter]
		if found {
			reporter.Reports++
		} else {
			reporter = ReportersReportCount{
				Power:   r.Power,
				Reports: 1,
				Height:  r.BlockNumber,
			}
		}
		reportersMap[r.Reporter] = reporter
		totalPower += r.Power
	}

	// Convert to sorted slice for deterministic iteration
	sortedReporters := make([]ReporterInfo, 0, len(reportersMap))
	for addr, data := range reportersMap {
		sortedReporters = append(sortedReporters, ReporterInfo{
			address: addr,
			data:    data,
		})
	}

	// Sort by address for deterministic ordering
	sort.Slice(sortedReporters, func(i, j int) bool {
		return sortedReporters[i].address < sortedReporters[j].address
	})

	// Process rewards in deterministic order
	totaldist := math.ZeroUint()
	for i, reporter := range sortedReporters {
		amount := CalculateRewardAmount(
			reporter.data.Power,
			reporter.data.Reports,
			totalPower,
			reward,
		)
		totaldist = totaldist.Add(amount.Value)

		reporterAddr, err := sdk.AccAddressFromBech32(reporter.address)
		if err != nil {
			return err
		}

		// Handle final reporter
		if i == len(sortedReporters)-1 {
			amount.Value = amount.Value.Add(math.NewUint(reward.Uint64()).MulUint64(1e6)).Sub(totaldist)
		}

		err = k.AllocateTip(ctx, reporterAddr.Bytes(), amount, reporter.data.Height)
		if err != nil {
			return err
		}
	}

	return k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		fromPool,
		reportertypes.TipsEscrowPool,
		sdk.NewCoins(sdk.NewCoin(layer.BondDenom, reward)),
	)
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
	norm_reward := math.NewUint(reward.Uint64())
	norm_reward = norm_reward.MulUint64(reporterPower).MulUint64(reportsCount).MulUint64(1e6)
	amount := norm_reward.Quo(math.NewUint(totalPower))
	return reportertypes.BigUint{Value: amount}
}

func (k Keeper) AllocateTip(ctx context.Context, addr []byte, amount reportertypes.BigUint, height uint64) error {
	return k.reporterKeeper.DivvyingTips(ctx, addr, amount, height)
}
