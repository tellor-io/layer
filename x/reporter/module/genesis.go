package reporter

import (
	"fmt"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
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

	exportModuleData(ctx, k)
	// this line is used by starport scaffolding # genesis/module/export
	return genesis
}

type ReporterStateEntry struct {
	ReporterAddr []byte
	Reporter     types.OracleReporter
}

type SelectorStateEntry struct {
	SelectorAddr []byte
	Selector     types.Selection
}

func exportModuleData(ctx sdk.Context, k keeper.Keeper) {
	writer, err := utils.NewModuleStateWriter("reporter_module_state.json")
	if err != nil {
		panic(err)
	}

	iterReporters, err := k.Reporters.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterReporters.Close()
	err = writer.StartArraySection("reporters", false)
	if err != nil {
		panic(err)
	}
	numReporters := 0
	for ; iterReporters.Valid(); iterReporters.Next() {
		reporter_addr, err := iterReporters.Key()
		if err != nil {
			panic(err)
		}
		reporter, err := iterReporters.Value()
		if err != nil {
			panic(err)
		}
		err = writer.WriteArrayItem(ReporterStateEntry{
			ReporterAddr: reporter_addr,
			Reporter:     reporter,
		})
		if err != nil {
			panic(err)
		}
		numReporters++
	}
	err = writer.EndArraySection(numReporters)
	if err != nil {
		panic(err)
	}

	iterSelectors, err := k.Selectors.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterSelectors.Close()
	err = writer.StartArraySection("selectors", false)
	if err != nil {
		panic(err)
	}
	numSelectors := 0
	for ; iterSelectors.Valid(); iterSelectors.Next() {
		selector_addr, err := iterSelectors.Key()
		if err != nil {
			panic(err)
		}
		selector, err := iterSelectors.Value()
		if err != nil {
			panic(err)
		}
		err = writer.WriteArrayItem(SelectorStateEntry{
			SelectorAddr: selector_addr,
			Selector:     selector,
		})
		if err != nil {
			panic(err)
		}
		numSelectors++
	}
	err = writer.EndArraySection(numSelectors)
	if err != nil {
		panic(err)
	}

	iterSelectorTips, err := k.SelectorTips.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterSelectorTips.Close()
	err = writer.StartArraySection("selector_tips", false)
	if err != nil {
		panic(err)
	}
	numSelectorTips := 0
	for ; iterSelectorTips.Valid(); iterSelectorTips.Next() {
		selector_addr, err := iterSelectorTips.Key()
		if err != nil {
			panic(err)
		}

		tips, err := iterSelectorTips.Value()
		if err != nil {
			panic(err)
		}
		err = writer.WriteArrayItem(types.SelectorTipsStateEntry{SelectorAddress: selector_addr, Tips: tips})
		if err != nil {
			panic(err)
		}
		numSelectorTips++
	}
	err = writer.EndArraySection(numSelectorTips)
	if err != nil {
		panic(err)
	}

	iterDisputedDelAmt, err := k.DisputedDelegationAmounts.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterDisputedDelAmt.Close()
	err = writer.StartArraySection("disputed_delegation_amounts", false)
	if err != nil {
		panic(err)
	}
	numDisputedDelAmts := 0
	for ; iterDisputedDelAmt.Valid(); iterDisputedDelAmt.Next() {
		disputeHashId, err := iterDisputedDelAmt.Key()
		if err != nil {
			panic(err)
		}

		delAmount, err := iterDisputedDelAmt.Value()
		if err != nil {
			panic(err)
		}
		err = writer.WriteArrayItem(types.DisputedDelegationAmountStateEntry{HashId: disputeHashId, DelegationAmount: &delAmount})
		if err != nil {
			panic(err)
		}
		numDisputedDelAmts++
	}
	err = writer.EndArraySection(numDisputedDelAmts)
	if err != nil {
		panic(err)
	}
	iterFeePaidFromStake, err := k.FeePaidFromStake.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	defer iterFeePaidFromStake.Close()
	err = writer.StartArraySection("fee_paid_from_stake", false)
	if err != nil {
		panic(err)
	}
	numFeePaidFromStake := 0
	for ; iterFeePaidFromStake.Valid(); iterFeePaidFromStake.Next() {
		disputeHashId, err := iterFeePaidFromStake.Key()
		if err != nil {
			panic(err)
		}

		delAmount, err := iterFeePaidFromStake.Value()
		if err != nil {
			panic(err)
		}
		err = writer.WriteArrayItem(types.FeePaidFromStakeStateEntry{HashId: disputeHashId, DelegationAmount: &delAmount})
		if err != nil {
			panic(err)
		}
		numFeePaidFromStake++
	}
	err = writer.EndArraySection(numFeePaidFromStake)
	if err != nil {
		panic(err)
	}
}
