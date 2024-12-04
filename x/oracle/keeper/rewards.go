package keeper

import (
	"context"
	"fmt"
	"sort"

	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ReportersReportCount struct {
	Power   uint64
	Reports uint64
	Height  uint64
	queryId []byte
}
type ReporterInfo struct {
	address string
	data    ReportersReportCount
}

// AllocateRewards distributes rewards to reporters based on their power and number of reports.
// It calculates the reward amount for each reporter and allocates the rewards.
// Finally, it sends the allocated rewards to the apprppopriate module based on the source of the reward.
func (k Keeper) AllocateRewards(ctx context.Context, report *types.Aggregate, reward math.Int, fromPool string) error {
	if reward.IsZero() {
		return nil
	}
	// Initialize totalPower to keep track of the total power of all reporters.
	totalPower := uint64(0)
	// First pass: collect data in map
	reportersMap := make(map[string]ReportersReportCount)
	// Use a struct to hold reporter info
	iter, err := k.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(report.MetaId, report.QueryId))
	if err != nil {
		fmt.Println("error getting reports", err)
		return err
	}

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		reporterk, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		report, err := k.Reports.Get(ctx, reporterk)
		if err != nil {
			return err
		}
		reporter, found := reportersMap[report.Reporter]
		if found {
			fmt.Println("found reporter", reporter)
			reporter.Reports++
		} else {
			reporter = ReportersReportCount{
				Power:   report.Power,
				Reports: 1,
				Height:  report.BlockNumber,
				queryId: report.QueryId,
			}
		}
		reportersMap[report.Reporter] = reporter
		totalPower += report.Power
	}
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
	return k.Allocate(ctx, sortedReporters, totalPower, reward, fromPool)
}

func (k Keeper) AllocateTBR(ctx context.Context, reports []types.Aggregate) error {
	if len(reports) == 0 {
		return nil
	}

	totalPower := uint64(0)
	reportersMap := make(map[string]ReportersReportCount)

	for _, aggRep := range reports {
		iter, err := k.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(aggRep.MetaId, aggRep.QueryId))
		if err != nil {
			return err
		}

		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			reporterk, err := iter.PrimaryKey()
			if err != nil {
				return err
			}
			report, err := k.Reports.Get(ctx, reporterk)
			if err != nil {
				return err
			}
			reporter, found := reportersMap[report.Reporter]
			if found {
				fmt.Println("found reporter", reporter)
				reporter.Reports++
			} else {
				reporter = ReportersReportCount{
					Power:   report.Power,
					Reports: 1,
					Height:  report.BlockNumber,
					queryId: report.QueryId,
				}
			}
			reportersMap[report.Reporter] = reporter
			totalPower += report.Power
		}
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
		tbr := k.GetTimeBasedRewards(ctx)
		err = k.Allocate(ctx, sortedReporters, totalPower, tbr, minttypes.TimeBasedRewards)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) Allocate(ctx context.Context, sortedReporters []ReporterInfo, totalPower uint64, reward math.Int, fromPool string) error {
	// Process rewards in deterministic order
	totaldist := math.LegacyZeroDec()
	for i, reporter := range sortedReporters {
		amount := CalculateRewardAmount(
			reporter.data.Power,
			reporter.data.Reports,
			totalPower,
			// reward is in loya
			reward,
		)
		totaldist = totaldist.Add(amount)

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
	return balance.Amount
}

func (k Keeper) GetTimeBasedRewardsAccount(ctx context.Context) sdk.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, minttypes.TimeBasedRewards)
}

func CalculateRewardAmount(reporterPower, reportsCount, totalPower uint64, reward math.Int) math.LegacyDec {
	rPower := math.LegacyNewDec(int64(reporterPower))
	rcount := math.LegacyNewDec(int64(reportsCount))
	tPower := math.LegacyNewDec(int64(totalPower))

	power := rPower.Mul(rcount)
	// reward is in loya
	// amount = (power/TotalPower) * reward
	amount := power.Quo(tPower).Mul(reward.ToLegacyDec())
	return amount
}

func (k Keeper) AllocateTip(ctx context.Context, addr, queryId []byte, amount math.LegacyDec, height uint64) error {
	return k.reporterKeeper.DivvyingTips(ctx, addr, amount, queryId, height)
}
