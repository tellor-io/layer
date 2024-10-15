package simulation

import (
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

func SimulateMsgUpdateSpec(
	txConfig client.TxConfig,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// use the default gov module account address as authority
		var authority sdk.AccAddress = address.Module("gov")

		queryTypeOptions := []string{"TWAP", "Mimicry", "EVMCall"}
		randomIndex := r.Intn(len(queryTypeOptions))
		queryType := queryTypeOptions[randomIndex]

		spec := types.DataSpec{
			DocumentHash:      "ipfs://bafkreig5war63asrzl6vygpqj3gca7nqcntxzoy4lxvu2wqtmwwxlqwzau",
			ResponseValueType: "string",
			AbiComponents: []*types.ABIComponent{
				{Name: "asset", FieldType: "string"},
				{Name: "currency", FieldType: "string"},
			},
			AggregationMethod: types.WeightedMedian,
			Registrar:         string(simAccount.Address),
			ReportBlockWindow: 4,
		}

		msg := &types.MsgUpdateDataSpec{
			QueryType: queryType,
			Spec:      spec,
		}

		// Pick a random number between 0 and 99
		// Half the time the authority will be wrong and tx should fail
		randomNum := r.Intn(100)
		if randomNum < 50 {
			fmt.Println("good authority")
			msg.Authority = authority.String()
		} else {
			msg.Authority = simAccount.Address.String()
			fmt.Println("bad authority")
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgRegisterSpec{}), "authority is not gov"), nil, nil
		}

		// Check if the spec exists
		// If spec does not exist, expect tx to fail
		specExists, _ := k.HasSpec(ctx, msg.QueryType)
		if !specExists {
			fmt.Println("spec does not exist yet")
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgRegisterSpec{}), "spec not registered yet"), nil, nil
		}

		// build full tx
		fees := sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(int64(simtypes.RandIntBetween(r, 50, 100)))))
		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         txConfig,
			Cdc:           nil,
			Msg:           msg,
			Context:       ctx,
			SimAccount:    simAccount,
			AccountKeeper: ak,
			ModuleName:    types.ModuleName,
		}
		return simulation.GenAndDeliverTx(txCtx, fees)
	}
}
