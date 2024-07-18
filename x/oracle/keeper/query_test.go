package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/oracle/keeper"
)

func TestNewQuerier(t *testing.T) {
	k, _, _, _, _, _ := testkeeper.OracleKeeper(t)
	q := keeper.NewQuerier(k)
	require.NotNil(t, q)
}
