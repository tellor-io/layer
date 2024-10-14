package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/tellor-io/layer/testutil"
)

func SimulateMsgSubmitValue(
	txConfig client.TxConfig,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		reporters, _ := k.AllReporters(ctx)
		if len(reporters) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgSubmitValue{}), "reporter not found"), nil, nil
		}
		reporterAdd := reporters[simtypes.RandIntBetween(r, 0, len(reporters))].Address

		hexValue := testutil.EncodeValue(r.Float64())
		qdata, _ := k.GetCurrentQueryInCycleList(ctx)
		msg := &types.MsgSubmitValue{
			Creator:   reporterAdd,
			QueryData: qdata,
			Value:     hexValue,
		}

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
