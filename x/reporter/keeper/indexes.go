package keeper

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

type ReporterDelegatorsIndex struct {
	Reporter *indexes.Multi[sdk.AccAddress, sdk.AccAddress, types.Delegation]
}

func (a ReporterDelegatorsIndex) IndexesList() []collections.Index[sdk.AccAddress, types.Delegation] {
	return []collections.Index[sdk.AccAddress, types.Delegation]{a.Reporter}
}

func NewDelegatorsIndex(sb *collections.SchemaBuilder) ReporterDelegatorsIndex {
	return ReporterDelegatorsIndex{
		Reporter: indexes.NewMulti(
			sb, types.ReporterDelegatorsIndexPrefix, "reporter_delegators_index",
			sdk.AccAddressKey, sdk.AccAddressKey,
			func(k sdk.AccAddress, del types.Delegation) (sdk.AccAddress, error) {
				return sdk.AccAddressFromBech32(del.Reporter)
			},
		),
	}
}
