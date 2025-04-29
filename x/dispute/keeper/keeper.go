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
		Disputes                           *collections.IndexedMap[uint64, types.Dispute, types.DisputesIndex]                           // key: dispute id
		Votes                              collections.Map[uint64, types.Vote]                                                           // key: dispute id
		Voter                              *collections.IndexedMap[collections.Pair[uint64, []byte], types.Voter, types.VotersVoteIndex] // key: dispute id + voter address
		ReportersWithDelegatorsVotedBefore collections.Map[collections.Pair[[]byte, uint64], math.Int]                                   // key: reporter address + dispute id
		BlockInfo                          collections.Map[[]byte, types.BlockInfo]                                                      // key: dispute.HashId
		DisputeFeePayer                    collections.Map[collections.Pair[uint64, []byte], types.PayerInfo]                            // key: dispute id + payer address
		// dust is extra tokens leftover after truncating decimals, stored as fixed256x12
		Dust              collections.Item[math.Int]
		VoteCountsByGroup collections.Map[uint64, types.StakeholderVoteCounts] // key: dispute id
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
		// maps dispute id to dispute, and indexes disputes by reporter, open status, and pending execution
		Disputes: collections.NewIndexedMap(sb, types.DisputesPrefix, "disputes", collections.Uint64Key, codec.CollValue[types.Dispute](cdc), types.NewDisputesIndex(sb)),
		// maps dispute id to vote
		Votes: collections.NewMap(sb, types.VotesPrefix, "votes", collections.Uint64Key, codec.CollValue[types.Vote](cdc)),
		// maps dispute id + voter address to voter's vote info and indexes voters by id, used for tallying votes
		Voter: collections.NewIndexedMap(sb,
			types.VoterVotePrefix,
			"voter_vote",
			collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey),
			codec.CollValue[types.Voter](cdc),
			types.NewVotersIndex(sb),
		),
		// maps reporter address + dispute id to reporter's stake - selectors' belonging to the reporter share that individually voted
		ReportersWithDelegatorsVotedBefore: collections.NewMap(sb,
			types.ReportersWithDelegatorsVotedBeforePrefix,
			"reporters_with_delegators_voted_before",
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			sdk.IntValue,
		),
		// maps dispute hash to block info, stores total reporting stake and total user tips at the time of dispute start
		BlockInfo: collections.NewMap(sb, types.BlockInfoPrefix, "block_info", collections.BytesKey, codec.CollValue[types.BlockInfo](cdc)),
		// maps dispute id + payer address to payer info, used to track who paid the dispute fee, how much and how (ie from stake or not)
		DisputeFeePayer: collections.NewMap(sb, types.DisputeFeePayerPrefix, "dispute_fee_payer", collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey), codec.CollValue[types.PayerInfo](cdc)),
		// a place to store dust fractions of tokens that remains during fee refunds, burned when they become whole amounts
		Dust: collections.NewItem(sb, types.DustKeyPrefix, "dust", sdk.IntValue),
		// maps dispute id to voter groups' vote counts, used to tally votes
		VoteCountsByGroup: collections.NewMap(sb, types.VoteCountsByGroupPrefix, "vote_counts_by_group", collections.Uint64Key, codec.CollValue[types.StakeholderVoteCounts](cdc)),
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
