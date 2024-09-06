package migrations

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type queryMetaIndex struct {
	HasReveals *indexes.Multi[bool, []byte, QueryMeta]
	QueryType  *indexes.Multi[string, []byte, QueryMeta]
}

func (a queryMetaIndex) IndexesList() []collections.Index[[]byte, QueryMeta] {
	return []collections.Index[[]byte, QueryMeta]{a.HasReveals, a.QueryType}
}

func NewQueryIndex(sb *collections.SchemaBuilder) queryMetaIndex {
	return queryMetaIndex{
		HasReveals: indexes.NewMulti(
			sb, types.QueryRevealedIdsIndexPrefix, "query_by_revealed",
			collections.BoolKey, collections.BytesKey,
			func(_ []byte, v QueryMeta) (bool, error) {
				return v.HasRevealedReports, nil
			},
		),
		QueryType: indexes.NewMulti(
			sb, types.QueryTypeIndexPrefix, "query_by_type",
			collections.StringKey, collections.BytesKey,
			func(_ []byte, v QueryMeta) (string, error) {
				return v.QueryType, nil
			},
		),
	}
}

type QueryMeta struct {
	Id                    uint64        `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Amount                math.Int      `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
	Expiration            time.Time     `protobuf:"bytes,3,opt,name=expiration,proto3,stdtime" json:"expiration"`
	RegistrySpecTimeframe time.Duration `protobuf:"bytes,4,opt,name=registry_spec_timeframe,json=registrySpecTimeframe,proto3,stdduration" json:"registry_spec_timeframe"`
	HasRevealedReports    bool          `protobuf:"varint,5,opt,name=has_revealed_reports,json=hasRevealedReports,proto3" json:"has_revealed_reports,omitempty"`
	QueryId               []byte        `protobuf:"bytes,6,opt,name=query_id,json=queryData,proto3" json:"query_id,omitempty"`
	QueryType             string        `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
	CycleList             bool          `protobuf:"varint,8,opt,name=cycle_list,json=cycleList,proto3" json:"cycle_list,omitempty"`
}

func (*QueryMeta) ProtoMessage()    {}
func (m *QueryMeta) Reset()         {}
func (m *QueryMeta) String() string { return "" }
func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec, newquery *collections.IndexedMap[collections.Pair[[]byte, uint64], types.QueryMeta, types.QueryMetaIndex]) error {
	sb := collections.NewSchemaBuilder(storeService)
	oldquery := collections.NewIndexedMap(sb,
		types.QueryTipPrefix,
		"query",
		collections.BytesKey,
		codec.CollValue[QueryMeta](cdc),
		NewQueryIndex(sb),
	)

	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()

	type query struct {
		key   collections.Pair[[]byte, uint64]
		value types.QueryMeta
	}
	type legacyquery struct {
		key   collections.Pair[[]byte, uint64]
		value QueryMeta
	}
	newqueries := make([]query, 0)
	oldqueries := make([]legacyquery, 0)

	err := oldquery.Walk(ctx, nil, func(key []byte, value QueryMeta) (bool, error) {
		oldqueries = append(oldqueries, legacyquery{key: collections.Join(key, value.Id), value: value})
		if !value.Expiration.Add(time.Second * 3).Before(blockTime) {
			newqueries = append(newqueries, query{key: collections.Join(key, value.Id), value: types.QueryMeta{
				Id:                    value.Id,
				Amount:                value.Amount,
				Expiration:            value.Expiration,
				RegistrySpecTimeframe: value.RegistrySpecTimeframe,
				HasRevealedReports:    value.HasRevealedReports,
				QueryData:             value.QueryId,
				QueryType:             value.QueryType,
				CycleList:             value.CycleList,
			}})
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	for _, q := range oldqueries {
		if err := oldquery.Remove(ctx, q.key.K1()); err != nil {
			return err
		}
	}
	for _, q := range newqueries {
		if err := newquery.Set(ctx, q.key, q.value); err != nil {
			return err
		}
	}
	return nil
}
