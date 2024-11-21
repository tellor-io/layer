package keeper

import (
	"context"
	"fmt"
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
	queryId []byte
}

// AllocateRewards distributes rewards to reporters based on their power and number of reports.
// It calculates the reward amount for each reporter and allocates the rewards.
// Finally, it sends the allocated rewards to the apprppopriate module based on the source of the reward.
func (k Keeper) AllocateRewards(ctx context.Context, reports []*types.Aggregate, reward math.Int, fromPool string) error {
	fmt.Println("(AllocateRewards) reward: ", reward)
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
	for _, report := range reports {
		for _, r := range report.Reporters {
			reporter, found := reportersMap[r.Reporter]
			if found {
				reporter.Reports++
			} else {
				reporter = ReportersReportCount{
					Power:   r.Power,
					Reports: 1,
					Height:  r.BlockNumber,
					queryId: report.QueryId,
				}
			}
			reportersMap[r.Reporter] = reporter
			totalPower += r.Power
		}
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
	totaldist := math.LegacyZeroDec()
	for i, reporter := range sortedReporters {
		fmt.Println("(AllocateRewards) reporter.data.Power: ", reporter.data.Power)
		fmt.Println("(AllocateRewards) reporter.data.Reports: ", reporter.data.Reports)
		fmt.Println("(AllocateRewards) totalPower: ", totalPower)
		fmt.Println("(AllocateRewards) reward: ", reward)
		amount := CalculateRewardAmount(
			reporter.data.Power,
			reporter.data.Reports,
			totalPower,
			// reward is in loya
			reward,
		)
		fmt.Println("(AllocateRewards) amount: ", amount)
		totaldist = totaldist.Add(amount)
		fmt.Println("(AllocateRewards) totaldist: ", totaldist)

		reporterAddr, err := sdk.AccAddressFromBech32(reporter.address)
		if err != nil {
			return err
		}

		// final reporter gets total reward - total distributed so far
		if i == len(sortedReporters)-1 {
			amount = amount.Add(math.LegacyNewDecFromInt(reward).Sub(totaldist))
		}

		err = k.AllocateTip(ctx, reporterAddr.Bytes(), reporter.data.queryId, amount, reporter.data.Height)
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
	fmt.Println("(GetTimeBasedRewards) balance: ", balance)
	return balance.Amount
}

func (k Keeper) GetTimeBasedRewardsAccount(ctx context.Context) sdk.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, minttypes.TimeBasedRewards)
}

func CalculateRewardAmount(reporterPower, reportsCount, totalPower uint64, reward math.Int)  math.LegacyDec {
	fmt.Println("(CalculateRewardAmount) reporterPower: ", reporterPower)
	fmt.Println("(CalculateRewardAmount) totalPower: ", totalPower)
	fmt.Println("(CalculateRewardAmount) reward: ", reward)
	power := math.LegacyNewDec(int64(reporterPower) * int64(reportsCount))
	// reward is in loya
	// amount = (power/TotalPower) * reward
	amount := power.Quo(math.LegacyNewDec(int64(totalPower))).Mul(math.LegacyNewDecFromInt(reward))
	fmt.Println("(CalculateRewardAmount) amount: ", amount)
	return amount
}

func (k Keeper) AllocateTip(ctx context.Context, addr, queryId []byte, amount math.LegacyDec, height uint64) error {
	fmt.Println("(AllocateTip) amount: ", amount)
	return k.reporterKeeper.DivvyingTips(ctx, addr, amount, queryId, height)
}
