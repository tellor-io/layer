package app_test

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestGroupDisabled asserts x/group is removed from the app (no handlers, registry
// types, or module). Group proposals would otherwise bypass ante stake caps.
func TestGroupDisabled(t *testing.T) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = app.DefaultNodeHome

	bApp := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)

	delegateURL := sdk.MsgTypeURL(&stakingtypes.MsgDelegate{})
	require.NotNil(t, bApp.MsgServiceRouter().HandlerByTypeURL(delegateURL))
	_, err := bApp.InterfaceRegistry().Resolve(delegateURL)
	require.NoError(t, err)
	require.NotNil(t, bApp.GetKey(stakingtypes.StoreKey))
	_, ok := bApp.ModuleManager().Modules[stakingtypes.ModuleName]
	require.True(t, ok)

	groupMsgs := []sdk.Msg{
		&group.MsgCreateGroup{},
		&group.MsgUpdateGroupMembers{},
		&group.MsgUpdateGroupAdmin{},
		&group.MsgUpdateGroupMetadata{},
		&group.MsgCreateGroupPolicy{},
		&group.MsgCreateGroupWithPolicy{},
		&group.MsgUpdateGroupPolicyAdmin{},
		&group.MsgUpdateGroupPolicyDecisionPolicy{},
		&group.MsgUpdateGroupPolicyMetadata{},
		&group.MsgSubmitProposal{},
		&group.MsgWithdrawProposal{},
		&group.MsgVote{},
		&group.MsgExec{},
		&group.MsgLeaveGroup{},
	}
	for _, msg := range groupMsgs {
		url := sdk.MsgTypeURL(msg)
		require.Nilf(t, bApp.MsgServiceRouter().HandlerByTypeURL(url), "group message %s must not be routable", url)
		_, err := bApp.InterfaceRegistry().Resolve(url)
		require.Errorf(t, err, "group message %s must not be resolvable", url)
	}

	// Store key stays mounted this release for StoreUpgrades.Deleted; drop it next upgrade.
	_, ok = bApp.ModuleManager().Modules[group.ModuleName]
	require.False(t, ok)
	require.NotNil(t, bApp.GetKey(group.StoreKey))
}
