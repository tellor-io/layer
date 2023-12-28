package simulation

import (
	"math/rand"
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	// "github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/mocks"
	r "github.com/tellor-io/layer/x/registry"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context
	// oracleKeeper   keeper.Keeper
	registryKeeper keeper.Keeper
	// stakingKeeper  *mocks.StakingKeeper
	accountKeeper *mocks.AccountKeeper
	queryClient   types.QueryClient
	msgServer     types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	require := s.Require()
	config := sdk.GetConfig()
	// set up account ? not needed ?
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)

	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(
		cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"RegistryParams",
	)
	s.registryKeeper = *keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
	)

	s.ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}
	r.InitGenesis(s.ctx, s.registryKeeper, genesisState)
	// Initialize params
	s.registryKeeper.SetParams(s.ctx, types.DefaultParams())
	s.msgServer = keeper.NewMsgServerImpl(s.registryKeeper)

}

func SimulateMsgRegisterSpec(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
	msgServer types.MsgServer,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		// create msg
		msg := &types.MsgRegisterSpec{
			Creator:   simAccount.Address.String(),
			QueryType: "testQueryType",
			Spec: types.DataSpec{
				DocumentHash: "testHash",
				ValueType:    "uint256",
			},
		}
		// register the spec
		_, err := msgServer.RegisterSpec(sdk.WrapSDKContext(ctx), msg)
		require.NoError(t, err)

		// return simtypes.NopMsg(types.ModuleName, msg.Type(), "RegisterSpec simulation not implemented"), nil, nil
		return simtypes.NewOperationMsg(msg, true, "", types.ModuleCdc), nil, err
	}
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
