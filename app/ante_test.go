package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/x/tx/signing"

	// antetestutil "cosmossdk.io/x/auth/ante/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/tellor-io/layer/app"
)

func TestNewAnteHandler(t *testing.T) {
	require := require.New(t)

	// nil account keeper
	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper: nil,
		},
	}
	anteHandler, err := app.NewAnteHandler(options)
	require.ErrorContains(err, "account keeper is required for ante builder")
	require.Nil(anteHandler)

	// nil bank keeper
	options = app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper: authkeeper.AccountKeeper{},
			BankKeeper:    nil,
		},
	}
	anteHandler, err = app.NewAnteHandler(options)
	require.ErrorContains(err, "bank keeper is required for ante builder")
	require.Nil(anteHandler)

	// nil SignModeHandler
	options = app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   authkeeper.AccountKeeper{},
			BankKeeper:      bankkeeper.BaseKeeper{},
			SignModeHandler: nil,
		},
	}
	anteHandler, err = app.NewAnteHandler(options)
	require.ErrorContains(err, "sign mode handler is required for ante builder")
	require.Nil(anteHandler)

	options = app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   authkeeper.AccountKeeper{},
			BankKeeper:      bankkeeper.BaseKeeper{},
			SignModeHandler: &signing.HandlerMap{},
		},
	}
	anteHandler, err = app.NewAnteHandler(options)
	require.NoError(err)
	require.NotNil(anteHandler)
}
