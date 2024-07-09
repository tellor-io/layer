package dispute

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/keeper"
)

func TestAutoCLIOptions(t *testing.T) {
	require := require.New(t)
	am := NewAppModule(nil, keeper.Keeper{}, nil, nil)

	moduleOptions := am.AutoCLIOptions()
	require.NotNil(moduleOptions)
}
