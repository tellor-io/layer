package keeper

import (
	"context"

	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	// "cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetTimeBasedRewards(ctx context.Context) math.Int {
	tbrAccount := k.GetTimeBasedRewardsAccount(ctx)
	balance := k.bankKeeper.GetBalance(ctx, tbrAccount.GetAddress(), layer.BondDenom)
	return balance.Amount
}

func (k Keeper) GetTimeBasedRewardsAccount(ctx context.Context) sdk.ModuleAccountI {
	return k.accountKeeper.GetModuleAccount(ctx, minttypes.TimeBasedRewards)
}

func (k Keeper) AllocateTip(ctx context.Context, addr, queryId []byte, amount math.LegacyDec, height uint64) error {
	return k.reporterKeeper.DivvyingTips(ctx, addr, amount, queryId, height)
}

func (k Keeper) DistributeRewards(ctx context.Context, tipRewardAllocation map[uint64]rewards, tipRewardKeys []uint64) error {
	for _, id := range tipRewardKeys {
		aggregateReward := tipRewardAllocation[id]
		aggregateReport := aggregateReward.aggregateReport
		iter, err := k.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(aggregateReport.MetaId, aggregateReport.QueryId))
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
			amount := math.LegacyNewDec(int64(report.Power)).Quo(math.LegacyNewDec(int64(aggregateReport.AggregatePower))).Mul(aggregateReward.reward)
			err = k.AllocateTip(ctx, reporterk.K2(), report.QueryId, amount, report.BlockNumber)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) DistributeTbr(
	ctx context.Context, tipRewardKeys []uint64,
	cyclelist []types.Aggregate, tipRewardAllocation map[uint64]rewards,
	totaltbrPower uint64, tbr math.Int,
) (map[uint64]rewards, []uint64) {
	totalPowerDec := math.LegacyNewDec(int64(totaltbrPower))
	for _, aggregateReport := range cyclelist {
		aggregatePower := math.LegacyNewDec(int64(aggregateReport.AggregatePower)).Quo(totalPowerDec)
		share := aggregatePower.Mul(tbr.ToLegacyDec())
		reward, ok := tipRewardAllocation[aggregateReport.MetaId]
		if !ok {
			tipRewardAllocation[aggregateReport.MetaId] = rewards{aggregateReport: aggregateReport, reward: share}
			tipRewardKeys = append(tipRewardKeys, aggregateReport.MetaId)
			continue
		}
		reward.reward = reward.reward.Add(share)
		tipRewardAllocation[aggregateReport.MetaId] = reward
	}
	return tipRewardAllocation, tipRewardKeys
}
