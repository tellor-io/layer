package simulation

import (
	"math/rand"

	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

func SimulateMsgRegisterSpec(
	txConfig client.TxConfig,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// create data spec
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
		// randomly pick one of these three query types to try to register
		// all of them get the same DataSpec for simplicity
		queryTypeOptions := []string{"TWAP", "Mimicry", "EVMCall"}
		randomIndex := r.Intn(len(queryTypeOptions))
		queryType := queryTypeOptions[randomIndex]

		// create Msg
		msg := &types.MsgRegisterSpec{
			Registrar: simAccount.Address.String(),
			QueryType: queryType,
			Spec:      spec,
		}

		// Check if the spec exists
		// If spec already exists, expect tx to fail
		specExists, _ := k.HasSpec(ctx, msg.QueryType)
		if specExists {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgRegisterSpec{}), "spec already registered"), nil, nil
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
