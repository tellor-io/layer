package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeWithQuerytype(t *testing.T) {
	querytype := "testQueryType"
	databytes := []byte("test")
	res, err := EncodeWithQuerytype(querytype, databytes)
	require.NoError(t, err)
	require.NotNil(t, res)
}
