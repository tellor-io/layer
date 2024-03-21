package types

import (
	context "context"
)

// combine multiple registry hooks, all hook functions are run in array sequence
var _ RegistryHooks = &MultiRegistryHooks{}

type MultiRegistryHooks []RegistryHooks

func NewMultiRegistryHooks(hooks ...RegistryHooks) MultiRegistryHooks {
	return hooks
}

func (h MultiRegistryHooks) AfterDataSpecUpdated(ctx context.Context, querytype string, dataspec DataSpec) error {
	for i := range h {
		if err := h[i].AfterDataSpecUpdated(ctx, querytype, dataspec); err != nil {
			return err
		}
	}

	return nil
}
