package simulation

import (
	"math/rand"

	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgWithdrawTip(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgWithdrawTip{
			SelectorAddress: simAccount.Address.String(),
		}

		// TODO: Handling the WithdrawTip simulation

		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "WithdrawTip simulation not implemented"), nil, nil
	}
}
