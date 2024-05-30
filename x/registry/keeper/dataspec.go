package keeper

import (
	"context"
	"strings"

	"github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetDataSpec sets the data specification for a given query type.
// It converts the query type to lowercase and then calls the Set method of the SpecRegistry to store the data specification.
func (k Keeper) SetDataSpec(ctx sdk.Context, querytype string, dataspec types.DataSpec) error {
	querytype = strings.ToLower(querytype)
	return k.SpecRegistry.Set(ctx, querytype, dataspec)
}

// GetSpec retrieves a DataSpec from the registry based on the provided query type.
// It converts the query type to lowercase before performing the retrieval.
// If the DataSpec is found, it is returned along with a nil error.
// If the DataSpec is not found, an empty DataSpec and an error are returned.
func (k Keeper) GetSpec(ctx context.Context, querytype string) (types.DataSpec, error) {
	querytype = strings.ToLower(querytype)
	return k.SpecRegistry.Get(ctx, querytype)
}

// HasSpec checks if a data specification with the given query type exists in the registry.
// It returns true if the data specification exists, otherwise false.
// It converts the query type parameter to lower case before, for keeping things consistent.
// Returns an error if there was an issue checking the registry.
func (k Keeper) HasSpec(ctx context.Context, querytype string) (bool, error) {
	querytype = strings.ToLower(querytype)
	return k.SpecRegistry.Has(ctx, querytype)
}
