package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParams_ParamKeyTable(t *testing.T) {
	require := require.New(t)

	table := ParamKeyTable()
	require.NotNil(table)
}

func TestParams_NewParams(t *testing.T) {
	require := require.New(t)

	params := NewParams()
	require.NotNil(params)
	require.Equal(params.MaxReportBufferWindow, DefaultMaxReportWindow)
}
