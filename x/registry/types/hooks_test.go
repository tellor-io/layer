package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNewMultiRegistryHooks(t *testing.T) {
	require := require.New(t)

	hooks := NewMultiRegistryHooks()
	require.NotNil(hooks)
}

func TestAfterDataSpecUpdated(t *testing.T) {
	require := require.New(t)

	hooks := NewMultiRegistryHooks()
	require.Nil(hooks)
	err := hooks.AfterDataSpecUpdated(sdk.Context{}, "", DataSpec{})
	require.NoError(err)
}
