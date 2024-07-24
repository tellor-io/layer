package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/x/tx/signing"

	// antetestutil "cosmossdk.io/x/auth/ante/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

func TestNewAnteHandler(t *testing.T) {
	require := require.New(t)

	// nil account keeper
	options := HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper: nil,
		},
	}
	anteHandler, err := NewAnteHandler(options)
	require.ErrorContains(err, "account keeper is required for ante builder")
	require.Nil(anteHandler)

	// nil bank keeper
	options = HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper: authkeeper.AccountKeeper{},
			BankKeeper:    nil,
		},
	}
	anteHandler, err = NewAnteHandler(options)
	require.ErrorContains(err, "bank keeper is required for ante builder")
	require.Nil(anteHandler)

	// nil SignModeHandler
	options = HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   authkeeper.AccountKeeper{},
			BankKeeper:      bankkeeper.BaseKeeper{},
			SignModeHandler: nil,
		},
	}
	anteHandler, err = NewAnteHandler(options)
	require.ErrorContains(err, "sign mode handler is required for ante builder")
	require.Nil(anteHandler)

	options = HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   authkeeper.AccountKeeper{},
			BankKeeper:      bankkeeper.BaseKeeper{},
			SignModeHandler: &signing.HandlerMap{},
		},
	}
	anteHandler, err = NewAnteHandler(options)
	require.NoError(err)
	require.NotNil(anteHandler)
}
