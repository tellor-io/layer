package integration_test

import (
	v6_1_6 "github.com/tellor-io/layer/app/upgrades/v6.1.6"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// Simulates a pre-upgrade dust send that lazily creates a BaseAccount at the
// deterministic tips_unlock_pool address. The upgrade helper must convert it
// without calling GetModuleAccount (which would panic).
func (s *IntegrationTestSuite) TestEnsureTipsUnlockPoolConvertsBaseAccount() {
	require := s.Require()
	ctx := s.Setup.Ctx
	ak := s.Setup.Accountkeeper

	addr := authtypes.NewModuleAddress(reportertypes.TipsUnlockPool)

	// Simulate a pre-upgrade dust send: a BaseAccount at the deterministic
	// module address (whether or not a module account was already in state).
	var accountNumber uint64
	if existing := ak.GetAccount(ctx, addr); existing != nil {
		accountNumber = existing.GetAccountNumber()
		ak.SetAccount(ctx, authtypes.NewBaseAccount(addr, nil, accountNumber, 0))
	} else {
		baseAcc := ak.NewAccount(ctx, authtypes.NewBaseAccountWithAddress(addr))
		accountNumber = baseAcc.GetAccountNumber()
		ak.SetAccount(ctx, baseAcc)
	}

	require.NotNil(ak.GetAccount(ctx, addr))
	_, isModule := ak.GetAccount(ctx, addr).(sdk.ModuleAccountI)
	require.False(isModule)

	require.PanicsWithValue("account is not a module account", func() {
		ak.GetModuleAccount(ctx, reportertypes.TipsUnlockPool)
	})

	require.NotPanics(func() {
		v6_1_6.EnsureTipsUnlockPoolModuleAccount(ctx, ak)
	})

	converted := ak.GetAccount(ctx, addr)
	require.NotNil(converted)
	macc, ok := converted.(sdk.ModuleAccountI)
	require.True(ok)
	require.Equal(reportertypes.TipsUnlockPool, macc.GetName())
	require.Equal(accountNumber, converted.GetAccountNumber())
	require.NotPanics(func() {
		_ = ak.GetModuleAccount(ctx, reportertypes.TipsUnlockPool)
	})
}

func (s *IntegrationTestSuite) TestEnsureTipsUnlockPoolIdempotentWhenAlreadyModuleAccount() {
	require := s.Require()
	ctx := s.Setup.Ctx
	ak := s.Setup.Accountkeeper

	before := ak.GetModuleAccount(ctx, reportertypes.TipsUnlockPool)
	require.NotNil(before)

	require.NotPanics(func() {
		v6_1_6.EnsureTipsUnlockPoolModuleAccount(ctx, ak)
	})

	after := ak.GetModuleAccount(ctx, reportertypes.TipsUnlockPool)
	require.NotNil(after)
	require.Equal(before.GetAddress().String(), after.GetAddress().String())
	require.Equal(before.GetAccountNumber(), after.GetAccountNumber())
}
