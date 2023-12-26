package simulation_test

import (
	"math/rand"
	"os"
	"testing"

	"cosmossdk.io/x/auth/simulation"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	"gotest.tools/v3/assert"
)

func TestRegisterSpec(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	logger := logger.NewLogfmtLogger(logger.NewSyncWriter(os.Stdout)) // ?
	header := types.Header                                            // ?
	ms := sdk.MultiStore(nil)                                         // ?

	ctx := sdk.NewContext(ms, header, true, logger)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalMsgs function
	weightedProposalMsgs := simulation.ProposalMsgs()
	assert.Assert(t, len(weightedProposalMsgs) == 1)

	w0 := weightedProposalMsgs[0]

	// tests w0 interface:
	assert.Equal(t, simulation.OpWeightMsgUpdateParams, w0.AppParamsKey())
	assert.Equal(t, simulation.DefaultWeightMsgUpdateParams, w0.DefaultWeight())

	msg := w0.MsgSimulatorFn()(r, ctx, accounts)
	msgUpdateParams, ok := msg.(*types.MsgUpdateParams)
	assert.Assert(t, ok)

	registrytypes.NewMsgRegisterSpec()

	assert.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgUpdateParams.Authority)
	assert.Assert(t, len(msgUpdateParams.Params.SendEnabled) == 0) //nolint:staticcheck // we're testing the old way here
	assert.Equal(t, true, msgUpdateParams.Params.DefaultSendEnabled)
}
