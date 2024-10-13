package simulation

import (
	"math/rand"

	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

func SimulateMsgCreateReporter(
	txConfig client.TxConfig,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		hasmin, err := k.HasMin(ctx, simAccount.Address, math.NewInt(100))
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgCreateReporter{}), "delegation not found"), nil, nil
		}

		if !hasmin {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgCreateReporter{}), "delegation is zero"), nil, nil
		}

		maxCommission := math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 0, 100)), 2)
		min := simtypes.RandIntBetween(r, 1_000_000, 2_000_000)

		reporterExists, _ := k.Reporters.Has(ctx, simAccount.Address)
		if reporterExists {
			return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(&types.MsgCreateReporter{}), "reporter already exists"), nil, nil
		}

		msg := &types.MsgCreateReporter{
			ReporterAddress:   simAccount.Address.String(),
			CommissionRate:    simtypes.RandomDecAmount(r, maxCommission),
			MinTokensRequired: math.NewInt(int64(min)),
		}

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

		return simulation.GenAndDeliverTx(txCtx, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(100))))
	}
}
