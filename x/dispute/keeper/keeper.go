package keeper

import (
	"context"
	"fmt"
	"time"

	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

		accountKeeper                      types.AccountKeeper
		bankKeeper                         types.BankKeeper
		oracleKeeper                       types.OracleKeeper
		reporterKeeper                     types.ReporterKeeper
		Disputes                           *collections.IndexedMap[uint64, types.Dispute, DisputesIndex] // dispute id -> dispute
		Votes                              collections.Map[uint64, types.Vote]
		Voter                              *collections.IndexedMap[collections.Pair[uint64, []byte], types.Voter, VotersVoteIndex]
		TeamVoter                          collections.Map[uint64, bool]
		UsersGroup                         collections.Map[collections.Pair[uint64, []byte], math.Int]
		ReportersGroup                     collections.Map[collections.Pair[uint64, []byte], math.Int]
		ReportersWithDelegatorsVotedBefore collections.Map[collections.Pair[[]byte, uint64], math.Int]
		BlockInfo                          collections.Map[[]byte, types.BlockInfo]
		DisputeFeePayer                    collections.Map[collections.Pair[uint64, []byte], types.PayerInfo]
		// dust is extra tokens leftover after truncating decimals, stored as fixed256x12
		Dust              collections.Item[math.Int]
		VoteCountsByGroup collections.Map[uint64, types.StakeholderVoteCounts]
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
		cdc:                                cdc,
		Params:                             collections.NewItem(sb, types.ParamsKeyPrefix(), "params", codec.CollValue[types.Params](cdc)),
		storeService:                       storeService,
		accountKeeper:                      accountKeeper,
		bankKeeper:                         bankKeeper,
		oracleKeeper:                       oracleKeeper,
		reporterKeeper:                     reporterKeeper,
		Disputes:                           collections.NewIndexedMap(sb, types.DisputesPrefix, "disputes", collections.Uint64Key, codec.CollValue[types.Dispute](cdc), NewDisputesIndex(sb)),
		Votes:                              collections.NewMap(sb, types.VotesPrefix, "votes", collections.Uint64Key, codec.CollValue[types.Vote](cdc)),
		Voter:                              collections.NewIndexedMap(sb, types.VoterVotePrefix, "voter_vote", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), codec.CollValue[types.Voter](cdc), NewVotersIndex(sb)),
		ReportersWithDelegatorsVotedBefore: collections.NewMap(sb, types.ReportersWithDelegatorsVotedBeforePrefix, "reporters_with_delegators_voted_before", collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key), sdk.IntValue),
		ReportersGroup:                     collections.NewMap(sb, types.ReporterPowerIndexPrefix, "reporters_group", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), sdk.IntValue),
		TeamVoter:                          collections.NewMap(sb, types.TeamVoterPrefix, "team_voter", collections.Uint64Key, collections.BoolValue),
		UsersGroup:                         collections.NewMap(sb, types.UsersGroupPrefix, "users_group", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), sdk.IntValue),
		BlockInfo:                          collections.NewMap(sb, types.BlockInfoPrefix, "block_info", collections.BytesKey, codec.CollValue[types.BlockInfo](cdc)),
		DisputeFeePayer:                    collections.NewMap(sb, types.DisputeFeePayerPrefix, "dispute_fee_payer", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), codec.CollValue[types.PayerInfo](cdc)),
		Dust:                               collections.NewItem(sb, types.DustKeyPrefix, "dust", sdk.IntValue),
		VoteCountsByGroup:                  collections.NewMap(sb, types.VoteCountsByGroupPrefix, "vote_counts_by_group", collections.Uint64Key, codec.CollValue[types.StakeholderVoteCounts](cdc)),
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
