package simulation

// import (
// 	"math/rand"

// 	"github.com/cosmos/cosmos-sdk/client"
// 	"github.com/cosmos/cosmos-sdk/codec"
// 	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
// 	"github.com/cosmos/cosmos-sdk/x/simulation"
// 	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
// 	"github.com/cosmos/cosmos-sdk/x/staking/types"
// 	registrytypes "github.com/tellor-io/layer/x/registry/types"
// )

// // simulation operation weight constants
// const (
// 	defaultWeightMsgRegisterSpec  int = 100
// 	defaultWeightMsgRegisterQuery int = 100

// 	OpWeightMsgRegisterSpec  = "op_weight_msg_register_spec"
// 	OpWeightMsgRegisterQuery = "op_weight_msg_register_query"
// )

// // WeightedOperations returns all the operations from the module with their respective weights
// func WeightedOperations(
// 	appParams simtypes.AppParams,
// 	cdc codec.JSONCodec,
// 	txGen client.TxConfig,
// 	ak types.AccountKeeper,
// 	bk types.BankKeeper,
// 	k *keeper.Keeper,
// 	msgServer registrytypes.MsgServer,
// ) simulation.WeightedOperations {
// 	var (
// 		weightMsgRegisterSpec  int
// 		weightMsgRegisterQuery int
// 	)

// 	appParams.GetOrGenerate(OpWeightMsgRegisterSpec, &weightMsgRegisterSpec, nil, func(_ *rand.Rand) {
// 		weightMsgRegisterSpec = defaultWeightMsgRegisterSpec
// 	})

// 	appParams.GetOrGenerate(OpWeightMsgRegisterQuery, &weightMsgRegisterQuery, nil, func(_ *rand.Rand), simtypes.ParamSimulator {
// 		weightMsgRegisterQuery == defaultWeightMsgRegisterQuery
// 	})

// 	return simulation.WeightedOperations{
// 		simulation.NewWeightedOperation(
// 			weightMsgRegisterSpec,
// 			SimulateMsgRegisterSpec(ak, bk, k, msgServer, nil),
// 		),
// 		simulation.NewWeightedOperation(
// 			weightMsgRegisterQuery,
// 			SimulateMsgRegisterQuery(ak, bk, k),
// 		),
// 	}
// }
