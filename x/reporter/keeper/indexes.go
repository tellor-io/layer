package keeper

import (
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
)

type ReporterSelectorsIndex struct {
	Reporter *indexes.Multi[[]byte, []byte, types.Selection]
}

func (a ReporterSelectorsIndex) IndexesList() []collections.Index[[]byte, types.Selection] {
	return []collections.Index[[]byte, types.Selection]{a.Reporter}
}

// maps a reporter address to its selectors' addresses
func NewSelectorsIndex(sb *collections.SchemaBuilder) ReporterSelectorsIndex {
	return ReporterSelectorsIndex{
		Reporter: indexes.NewMulti(
			sb, types.ReporterSelectorsIndexPrefix, "reporter_selectors_index",
			collections.BytesKey, collections.BytesKey,
			func(k []byte, del types.Selection) ([]byte, error) {
				return del.Reporter, nil
			},
		),
	}
}
