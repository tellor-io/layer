package reporter

import (
	"fmt"

	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
	fmt.Println("set params: ", genState.Params)
	c := ctx.BlockTime()
	err = k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: &c,
		Amount:     math.ZeroInt(),
	})
	if err != nil {
		panic(err)
	}

	// loop through selector tips found in genesis file and add them to state
	for _, data := range genState.SelectorTips {
		fmt.Println("adding selector tip", data.SelectorAddress, data.Tips)
		err = k.SelectorTips.Set(ctx, data.SelectorAddress, data.Tips)
		if err != nil {
			panic(err)
		}
	}

	// loop through DisputedDelegationAmounts found in the genesis file and add them to state
	for _, data := range genState.DisputedDelegationAmounts {
		err = k.DisputedDelegationAmounts.Set(ctx, data.HashId, *data.DelegationAmount)
		if err != nil {
			panic(err)
		}
	}

	// loop through FeePaidFromStake found in the genesis and add to state
	for _, data := range genState.FeePaidFromStake {
		err = k.FeePaidFromStake.Set(ctx, data.HashId, *data.DelegationAmount)
		if err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Params = params

	// iterSelectorTips, err := k.SelectorTips.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	// if err != nil {
	// 	panic(err)
	// }
	// selectorTips := make([]*types.SelectorTipsStateEntry, 0)
	// for ; iterSelectorTips.Valid(); iterSelectorTips.Next() {
	// 	selector_addr, err := iterSelectorTips.Key()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	tips, err := iterSelectorTips.Value()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	selectorTips = append(selectorTips, &types.SelectorTipsStateEntry{SelectorAddress: selector_addr, Tips: tips})
	// }
	// genesis.SelectorTips = selectorTips

	// iterDisputedDelAmt, err := k.DisputedDelegationAmounts.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	// if err != nil {
	// 	panic(err)
	// }
	// disputedDelAmts := make([]*types.DisputedDelegationAmountStateEntry, 0)
	// for ; iterDisputedDelAmt.Valid(); iterDisputedDelAmt.Next() {
	// 	disputeHashId, err := iterDisputedDelAmt.Key()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	delAmount, err := iterDisputedDelAmt.Value()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	disputedDelAmts = append(disputedDelAmts, &types.DisputedDelegationAmountStateEntry{HashId: disputeHashId, DelegationAmount: &delAmount})
	// }
	// genesis.DisputedDelegationAmounts = disputedDelAmts

	// iterFeePaidFromStake, err := k.FeePaidFromStake.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	// if err != nil {
	// 	panic(err)
	// }
	// feePaidFromStake := make([]*types.FeePaidFromStakeStateEntry, 0)
	// for ; iterFeePaidFromStake.Valid(); iterFeePaidFromStake.Next() {
	// 	disputeHashId, err := iterFeePaidFromStake.Key()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	delAmount, err := iterFeePaidFromStake.Value()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	feePaidFromStake = append(feePaidFromStake, &types.FeePaidFromStakeStateEntry{HashId: disputeHashId, DelegationAmount: &delAmount})
	// }
	// genesis.FeePaidFromStake = feePaidFromStake
	// this line is used by starport scaffolding # genesis/module/export
	return genesis
}
