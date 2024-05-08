package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"github.com/tellor-io/layer/x/dispute/types"
)

const (
	ONE_DAY    = 24 * time.Hour
	TWO_DAYS   = 2 * 24 * time.Hour
	THREE_DAYS = 3 * 24 * time.Hour
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		Params       collections.Item[types.Params]

		accountKeeper  types.AccountKeeper
		bankKeeper     types.BankKeeper
		oracleKeeper   types.OracleKeeper
		reporterKeeper types.ReporterKeeper
		Disputes       *collections.IndexedMap[uint64, types.Dispute, DisputesIndex] // dispute id -> dispute
		OpenDisputes   collections.Item[types.OpenDisputes]
		Votes          collections.Map[uint64, types.Vote]
		Voter          *collections.IndexedMap[collections.Pair[uint64, sdk.AccAddress], types.Voter, VotersVoteIndex]
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	reporterKeeper types.ReporterKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	return Keeper{
		cdc:            cdc,
		Params:         collections.NewItem(sb, types.ParamsKeyPrefix(), "params", codec.CollValue[types.Params](cdc)),
		storeService:   storeService,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		oracleKeeper:   oracleKeeper,
		reporterKeeper: reporterKeeper,
		Disputes:       collections.NewIndexedMap(sb, types.DisputesPrefix, "disputes", collections.Uint64Key, codec.CollValue[types.Dispute](cdc), NewDisputesIndex(sb)),
		OpenDisputes:   collections.NewItem(sb, types.OpenDisputeIdsPrefix, "open_disputes", codec.CollValue[types.OpenDisputes](cdc)),
		Votes:          collections.NewMap(sb, types.VotesPrefix, "votes", collections.Uint64Key, codec.CollValue[types.Vote](cdc)),
		Voter:          collections.NewIndexedMap(sb, types.VoterVotePrefix, "voter_vote", collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey), codec.CollValue[types.Voter](cdc), NewVotersIndex(sb)),
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
