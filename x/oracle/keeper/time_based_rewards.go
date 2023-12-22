package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k Keeper) AllocateTimeBasedRewards(ctx sdk.Context, report *types.Aggregate) {
	totalPower := k.stakingKeeper.GetLastTotalPower(ctx)
	totalReporterPower := int64(0)
	for _, reporter := range report.Reporters {
		totalReporterPower += reporter.Power
	}

	rewardPercentage := math.LegacyNewDec(totalReporterPower).QuoInt(totalPower)
	rewardPool := k.distrKeeper.GetFeePoolCommunityCoins(ctx)
	tbr := rewardPool.MulDecTruncate(rewardPercentage)
	remaining := tbr
	for _, r := range report.Reporters {
		validator := k.stakingKeeper.Validator(ctx, sdk.ValAddress(sdk.MustAccAddressFromBech32(r.Reporter)))
		powerFraction := math.LegacyNewDec(r.Power).QuoTruncate(math.LegacyNewDec(totalReporterPower))
		reward := tbr.MulDecTruncate(powerFraction)
		k.distrKeeper.AllocateTokensToValidator(ctx, validator, reward)
		remaining = remaining.Sub(reward)
	}

	feePool := k.distrKeeper.GetFeePool(ctx)
	feePool.CommunityPool = rewardPool.Sub(tbr).Add(remaining...)

	k.distrKeeper.SetFeePool(ctx, feePool)

}
