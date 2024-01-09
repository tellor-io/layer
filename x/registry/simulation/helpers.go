package simulation

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

// FindAccount find a specific address from an account list
func FindAccount(accs []simtypes.Account, address string) (simtypes.Account, bool) {
	creator, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return simtypes.FindAccount(accs, creator)
}

// func setupMsgServer(t testing.TB) (types.MsgServer, context.Context, keeper.Keeper, storeTypes.KVStoreKey) {
// 	k, ctx, key := keepertest.RegistryKeeper(t)
// 	// returns a MsgServerImpl struct, sdk.Context, keeper.Keeper, and storeTypes.KVStoreKey
// 	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx), *k, key
// }

func SimulateMsgRegisterSpec(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		// create msgServer
		msgServer := keeper.NewMsgServerImpl(k) //
		fmt.Println("rand.Uint64:   ", rand.Uint64())
		randomNumber := rand.Uint64()
		randomQueryType := strconv.FormatUint(randomNumber, 10)
		randomDocHash := strconv.FormatUint((randomNumber + 1), 10)

		// create msg
		msg := &types.MsgRegisterSpec{
			Creator:   simAccount.Address.String(),
			QueryType: randomQueryType,
			Spec: types.DataSpec{
				DocumentHash: randomDocHash,
				ValueType:    "uint256",
			},
		}
		fmt.Println("msg: ", msg)
		// register the spec
		fmt.Println("ctx: ", ctx)
		fmt.Println("ctx.Context()", ctx.Context())

		_, err := msgServer.RegisterSpec(ctx, msg)
		// require.NoError(t, err)

		// return simtypes.NopMsg(types.ModuleName, msg.Type(), "RegisterSpec simulation not implemented"), nil, nil
		return simtypes.NewOperationMsg(msg, true, "", types.ModuleCdc), nil, err
	}
}
