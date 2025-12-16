package keeper

import (
	"context"

	layer "github.com/tellor-io/layer/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
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

func (k Keeper) AllocateTBR(ctx context.Context, addr []byte, amount math.LegacyDec) error {
	return k.reporterKeeper.DivvyingTips(ctx, addr, amount)
}

func (k Keeper) DistributeTip(ctx context.Context, aggregateReport types.Aggregate, reward math.LegacyDec) error {
	iter, err := k.Reports.Indexes.IdQueryId.MatchExact(ctx, collections.Join(aggregateReport.MetaId, aggregateReport.QueryId))
	if err != nil {
		return err
	}

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		reportKey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		report, err := k.Reports.Get(ctx, reportKey)
		if err != nil {
			return err
		}
		reporter := reportKey.K2()
		amount := math.LegacyNewDec(int64(report.Power)).Quo(math.LegacyNewDec(int64(aggregateReport.AggregatePower))).Mul(reward)
		err = k.reporterKeeper.DivvyingTips(ctx, reporter, amount)
		if err != nil {
			return err
		}

		// track liveness for cyclelist reports
		if report.Cyclelist {
			if err := k.UpdateReporterLiveness(ctx, reporter, report.QueryId, report.Power); err != nil {
				return err
			}
		}
	}
	return nil
}
