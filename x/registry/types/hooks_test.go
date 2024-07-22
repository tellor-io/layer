package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
