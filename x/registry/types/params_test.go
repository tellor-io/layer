package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParams_ParamKeyTable(t *testing.T) {
	table := ParamKeyTable()
	require.NotNil(t, table)
}
