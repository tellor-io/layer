package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	"github.com/tellor-io/layer/x/reporter/types"
)

type ReporterDelegatorsIndex struct {
	Reporter *indexes.Multi[[]byte, []byte, types.Delegation]
}

func (a ReporterDelegatorsIndex) IndexesList() []collections.Index[[]byte, types.Delegation] {
	return []collections.Index[[]byte, types.Delegation]{a.Reporter}
}

func NewDelegatorsIndex(sb *collections.SchemaBuilder) ReporterDelegatorsIndex {
	return ReporterDelegatorsIndex{
		Reporter: indexes.NewMulti(
			sb, types.ReporterDelegatorsIndexPrefix, "reporter_delegators_index",
			collections.BytesKey, collections.BytesKey,
			func(k []byte, del types.Delegation) ([]byte, error) {
				return del.Reporter, nil
			},
		),
	}
}
