package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"layer/x/oracle/keeper"
	"layer/x/oracle/types"
)

func SimulateMsgSubmitValue(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgSubmitValue{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the SubmitValue simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "SubmitValue simulation not implemented"), nil, nil
	}
}
