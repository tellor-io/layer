package ante

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/encoding"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func mustAny(msg sdk.Msg) *codectypes.Any {
	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}
	return any
}

func buildTx(t *testing.T, msgs ...sdk.Msg) sdk.Tx {
	t.Helper()

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(msgs...))
	return txBuilder.GetTx()
}

// nestedExec wraps a message in authz exec messages to exercise recursion limits.
func nestedExec(depth int, msg sdk.Msg) sdk.Msg {
	wrapped := msg
	for i := 0; i < depth; i++ {
		wrapped = &authz.MsgExec{
			Grantee: sample.AccAddressBytes().String(),
			Msgs:    []*codectypes.Any{mustAny(wrapped)},
		}
	}
	return wrapped
}

// coin keeps staking-message setup compact while still showing the loya amount.
func coin(amount int64) sdk.Coin {
	return sdk.Coin{Denom: "loya", Amount: math.NewInt(amount)}
}

// validator creates the staking records used by ante tests; shares match tokens unless a test overrides them.
func validator(addr sdk.ValAddress, status stakingtypes.BondStatus, tokens math.Int) stakingtypes.Validator {
	return stakingtypes.Validator{
		OperatorAddress:   addr.String(),
		Status:            status,
		Tokens:            tokens,
		DelegatorShares:   tokens.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
}

// delegation ties a delegator to a validator with shares that represent the requested token amount.
func delegation(delegator sdk.AccAddress, validator sdk.ValAddress, tokens math.Int) stakingtypes.Delegation {
	return stakingtypes.Delegation{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: validator.String(),
		Shares:           tokens.ToLegacyDec(),
	}
}

// valAddress gives replacement tests stable address ordering for power ties.
func valAddress(fill byte) sdk.ValAddress {
	addr := make([]byte, 20)
	for i := range addr {
		addr[i] = fill
	}
	return sdk.ValAddress(addr)
}

// mockValidator registers a validator lookup without call-count coupling.
func mockValidator(sk *mocks.StakingKeeper, ctx sdk.Context, val stakingtypes.Validator) {
	valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
	if err != nil {
		panic(err)
	}
	sk.On("GetValidator", ctx, valAddr).Return(val, nil)
}

// mockPowerStore makes the active-set simulation deterministic.
func mockPowerStore(sk *mocks.StakingKeeper, ctx sdk.Context, maxValidators uint32, vals ...sdk.ValAddress) {
	mockPowerStoreWithReduction(sk, ctx, maxValidators, math.OneInt(), vals...)
}

// mockPowerStoreWithReduction lets tests exercise consensus-power truncation.
func mockPowerStoreWithReduction(sk *mocks.StakingKeeper, ctx sdk.Context, maxValidators uint32, powerReduction math.Int, vals ...sdk.ValAddress) {
	values := make([][]byte, 0, len(vals))
	for _, val := range vals {
		values = append(values, val)
	}
	sk.On("MaxValidators", ctx).Return(maxValidators, nil)
	sk.On("PowerReduction", ctx).Return(powerReduction)
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: values}, nil)
}

func mockIterateDelegations(sk *mocks.StakingKeeper, ctx sdk.Context, delegator sdk.AccAddress, delegations []stakingtypes.Delegation) {
	sk.On("IterateDelegatorDelegations", ctx, delegator, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for _, delegation := range delegations {
			if fn(delegation) {
				return
			}
		}
	})
}

func TestTrackStakeChanges(t *testing.T) {
	delAddr := sample.AccAddressBytes()
	srcValAddr := sdk.ValAddress(sample.AccAddressBytes())
	dstValAddr := sdk.ValAddress(sample.AccAddressBytes())
	fivePercentErr := "total stake increase exceeds the allowed 5% threshold within a twelve-hour period"
	decreaseErr := "total stake decrease exceeds the allowed 5% threshold within a twelve-hour period"
	nestedErr := fmt.Sprintf("nested message count exceeds the maximum allowed: Limit is %d", MaxNestedMsgCount)

	testCases := []struct {
		name    string
		msg     sdk.Msg
		wantErr error
		wantMsg string
		setup   func(*mocks.StakingKeeper, sdk.Context)
	}{
		{
			name: "delegate ok",
			msg: &stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: srcValAddr.String(),
				Amount:           coin(1),
			},
			setup: func(sk *mocks.StakingKeeper, ctx sdk.Context) {
				mockValidator(sk, ctx, validator(srcValAddr, stakingtypes.Bonded, math.NewInt(100)))
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{}, nil)
				mockIterateDelegations(sk, ctx, delAddr, []stakingtypes.Delegation{})
			},
		},
		{
			name: "max delegations",
			msg: &stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: srcValAddr.String(),
				Amount:           coin(1),
			},
			wantErr: types.ErrExceedsMaxDelegations,
			setup: func(sk *mocks.StakingKeeper, ctx sdk.Context) {
				mockValidator(sk, ctx, validator(srcValAddr, stakingtypes.Bonded, math.NewInt(100)))
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, nil)
			},
		},
		{
			name: "cancel over 5",
			msg: &stakingtypes.MsgCancelUnbondingDelegation{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: srcValAddr.String(),
				Amount:           coin(100),
			},
			wantMsg: fivePercentErr,
			setup: func(sk *mocks.StakingKeeper, ctx sdk.Context) {
				mockValidator(sk, ctx, validator(srcValAddr, stakingtypes.Bonded, math.NewInt(100)))
			},
		},
		{
			name: "undelegate over 5",
			msg: &stakingtypes.MsgUndelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: srcValAddr.String(),
				Amount:           coin(95),
			},
			wantMsg: decreaseErr,
			setup: func(sk *mocks.StakingKeeper, ctx sdk.Context) {
				mockValidator(sk, ctx, validator(srcValAddr, stakingtypes.Bonded, math.NewInt(100)))
			},
		},
		{
			name: "other msg",
			msg: &types.MsgUpdateParams{
				Authority: sample.AccAddressBytes().String(),
				Params:    types.Params{},
			},
		},
		{
			name: "empty authz",
			msg:  &authz.MsgExec{},
		},
		{
			name: "authz over 5",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&stakingtypes.MsgCancelUnbondingDelegation{
						DelegatorAddress: delAddr.String(),
						ValidatorAddress: srcValAddr.String(),
						Amount:           coin(100),
					}),
				},
			},
			wantMsg: fivePercentErr,
			setup: func(sk *mocks.StakingKeeper, ctx sdk.Context) {
				mockValidator(sk, ctx, validator(srcValAddr, stakingtypes.Bonded, math.NewInt(100)))
			},
		},
		{
			name: "nested limit",
			msg: nestedExec(MaxNestedMsgCount, &stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: dstValAddr.String(),
				Amount:           coin(1),
			}),
			wantMsg: nestedErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			decorator := NewTrackStakeChangesDecorator(k, sk)
			require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: math.NewInt(100)}))
			sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
			if tc.setup != nil {
				tc.setup(sk, ctx)
			}

			// Each case isolates one ante decision so failures point at the violated rule.
			_, err := decorator.AnteHandle(ctx, buildTx(t, tc.msg), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				return ctx, nil
			})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			if tc.wantMsg != "" {
				require.ErrorContains(t, err, tc.wantMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestShareCap(t *testing.T) {
	testCases := []struct {
		name          string
		blockHeight   int64
		currentTotal  math.Int
		existingStake math.Int
		delegateAmts  []math.Int
		err           error
	}{
		{
			name:          "blocks delegate over 30 percent",
			blockHeight:   1,
			currentTotal:  math.NewInt(100),
			existingStake: math.NewInt(30),
			delegateAmts:  []math.Int{math.OneInt()},
			err:           types.ErrExceedsMaxStakeShare,
		},
		{
			name:          "allows exactly 30 percent",
			blockHeight:   1,
			currentTotal:  math.NewInt(99),
			existingStake: math.NewInt(29),
			delegateAmts:  []math.Int{math.OneInt()},
			err:           nil,
		},
		{
			name:          "tracks multiple delegate messages in one tx",
			blockHeight:   1,
			currentTotal:  math.NewInt(100),
			existingStake: math.NewInt(29),
			delegateAmts:  []math.Int{math.OneInt(), math.OneInt()},
			err:           types.ErrExceedsMaxStakeShare,
		},
		{
			name:          "allows genesis gentx bootstrap at height zero",
			blockHeight:   0,
			currentTotal:  math.NewInt(100),
			existingStake: math.NewInt(30),
			delegateAmts:  []math.Int{math.OneInt()},
			err:           nil,
		},
	}

	s := encoding.GetTestEncodingCfg()
	clientCtx := client.Context{}.
		WithTxConfig(s.TxConfig)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			ctx = ctx.WithBlockHeight(tc.blockHeight)
			decorator := NewTrackStakeChangesDecorator(k, sk)
			require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
				Expiration: nil,
				Amount:     tc.currentTotal,
			}))

			delAddr := sample.AccAddressBytes()
			valAddr := sdk.ValAddress(sample.AccAddressBytes())
			validator := stakingtypes.Validator{
				OperatorAddress:   valAddr.String(),
				Status:            stakingtypes.Bonded,
				Tokens:            tc.existingStake,
				DelegatorShares:   tc.existingStake.ToLegacyDec(),
				MinSelfDelegation: math.OneInt(),
			}
			delegations := []stakingtypes.Delegation{
				{
					DelegatorAddress: delAddr.String(),
					ValidatorAddress: valAddr.String(),
					Shares:           tc.existingStake.ToLegacyDec(),
				},
			}

			sk.On("TotalBondedTokens", ctx).Return(tc.currentTotal, nil)
			sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return(delegations, nil)
			sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
			sk.On("IterateDelegatorDelegations", ctx, delAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
				fn := args.Get(2).(func(stakingtypes.Delegation) bool)
				for _, delegation := range delegations {
					if fn(delegation) {
						return
					}
				}
			})

			msgs := make([]sdk.Msg, 0, len(tc.delegateAmts))
			for _, amount := range tc.delegateAmts {
				msgs = append(msgs, &stakingtypes.MsgDelegate{
					DelegatorAddress: delAddr.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.Coin{Denom: "loya", Amount: amount},
				})
			}
			txBuilder := clientCtx.TxConfig.NewTxBuilder()
			require.NoError(t, txBuilder.SetMsgs(msgs...))
			tx := txBuilder.GetTx()

			_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
				return ctx, nil
			})

			// Height 0 is InitChain gentx replay, where validators are bonded
			// one at a time. Post-genesis blocks enforce the 30% delegator cap
			// against the final stake produced by all messages in the tx.
			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShareCapTxDrop(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	attackerAddr := sample.AccAddressBytes()
	attackerValAddr := sdk.ValAddress(sample.AccAddressBytes())
	attackerValidator := stakingtypes.Validator{
		OperatorAddress:   attackerValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(29),
		DelegatorShares:   math.NewInt(29).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	attackerDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: attackerValAddr.String(),
			Shares:           math.NewInt(29).ToLegacyDec(),
		},
	}
	otherAddr := sample.AccAddressBytes()
	otherValAddr := sdk.ValAddress(sample.AccAddressBytes())
	otherValidator := stakingtypes.Validator{
		OperatorAddress:   otherValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(10),
		DelegatorShares:   math.NewInt(10).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, attackerAddr).Return(attackerDelegations, nil)
	sk.On("GetValidator", ctx, attackerValAddr).Return(attackerValidator, nil)
	sk.On("GetValidator", ctx, otherValAddr).Return(otherValidator, nil)
	mockPowerStore(sk, ctx, 2, attackerValAddr, otherValAddr)
	sk.On("IterateDelegatorDelegations", ctx, attackerAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for _, delegation := range attackerDelegations {
			if fn(delegation) {
				return
			}
		}
	})

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	// The tx first removes 5 tokens from another delegator, lowering total
	// bonded stake, then adds 1 token to the attacker. The final state is
	// attacker 30 / total 96, which is above the 30% cap and must fail.
	require.NoError(t, txBuilder.SetMsgs(
		&stakingtypes.MsgUndelegate{
			DelegatorAddress: otherAddr.String(),
			ValidatorAddress: otherValAddr.String(),
			Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(5)},
		},
		&stakingtypes.MsgDelegate{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: attackerValAddr.String(),
			Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
		},
	))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapReduction(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	validator := stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(40),
		DelegatorShares:   math.NewInt(40).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	mockPowerStore(sk, ctx, 1, valAddr)

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	// A delegator already over the cap must still be able to reduce stake.
	// This tx only undelegates, so it cannot increase concentration risk.
	require.NoError(t, txBuilder.SetMsgs(&stakingtypes.MsgUndelegate{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.NoError(t, err)
}

func TestShareCapFinalTotal(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	aliceAddr := sample.AccAddressBytes()
	aliceValAddr := sdk.ValAddress(sample.AccAddressBytes())
	aliceValidator := stakingtypes.Validator{
		OperatorAddress:   aliceValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(29),
		DelegatorShares:   math.NewInt(29).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	aliceDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: aliceAddr.String(),
			ValidatorAddress: aliceValAddr.String(),
			Shares:           math.NewInt(29).ToLegacyDec(),
		},
	}

	bobAddr := sample.AccAddressBytes()
	bobValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bobValidator := stakingtypes.Validator{
		OperatorAddress:   bobValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(10),
		DelegatorShares:   math.NewInt(10).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, aliceAddr).Return(aliceDelegations, nil)
	sk.On("GetValidator", ctx, aliceValAddr).Return(aliceValidator, nil)
	sk.On("GetValidator", ctx, bobValAddr).Return(bobValidator, nil)
	mockPowerStore(sk, ctx, 2, aliceValAddr, bobValAddr)
	sk.On("IterateDelegatorDelegations", ctx, aliceAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for _, delegation := range aliceDelegations {
			if fn(delegation) {
				return
			}
		}
	})

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	// Alice would be exactly 30 / 100 after her delegate alone. Bob's
	// undelegate in the same tx lowers final total stake to 98, so Alice's
	// final share exceeds 30% and the whole tx must fail.
	require.NoError(t, txBuilder.SetMsgs(
		&stakingtypes.MsgDelegate{
			DelegatorAddress: aliceAddr.String(),
			ValidatorAddress: aliceValAddr.String(),
			Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
		},
		&stakingtypes.MsgUndelegate{
			DelegatorAddress: bobAddr.String(),
			ValidatorAddress: bobValAddr.String(),
			Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(2)},
		},
	))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapPortfolio(t *testing.T) {
	testCases := []struct {
		name        string
		delegateAmt math.Int
		err         error
	}{
		{
			name:        "allows exact cap",
			delegateAmt: math.OneInt(),
			err:         nil,
		},
		{
			name:        "blocks aggregate bonded stake over cap",
			delegateAmt: math.NewInt(2),
			err:         types.ErrExceedsMaxStakeShare,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
			ctx = ctx.WithBlockHeight(1)
			decorator := NewTrackStakeChangesDecorator(k, sk)
			currentTotal := math.NewInt(99)
			require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
				Expiration: nil,
				Amount:     currentTotal,
			}))

			delAddr := sample.AccAddressBytes()
			valAAddr := sdk.ValAddress(sample.AccAddressBytes())
			valBAddr := sdk.ValAddress(sample.AccAddressBytes())
			unbondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
			valA := stakingtypes.Validator{
				OperatorAddress:   valAAddr.String(),
				Status:            stakingtypes.Bonded,
				Tokens:            math.NewInt(20),
				DelegatorShares:   math.NewInt(20).ToLegacyDec(),
				MinSelfDelegation: math.OneInt(),
			}
			valB := stakingtypes.Validator{
				OperatorAddress:   valBAddr.String(),
				Status:            stakingtypes.Bonded,
				Tokens:            math.NewInt(9),
				DelegatorShares:   math.NewInt(9).ToLegacyDec(),
				MinSelfDelegation: math.OneInt(),
			}
			unbondedVal := stakingtypes.Validator{
				OperatorAddress:   unbondedValAddr.String(),
				Status:            stakingtypes.Unbonded,
				Tokens:            math.NewInt(100),
				DelegatorShares:   math.NewInt(100).ToLegacyDec(),
				MinSelfDelegation: math.OneInt(),
			}
			delegations := []stakingtypes.Delegation{
				{
					DelegatorAddress: delAddr.String(),
					ValidatorAddress: valAAddr.String(),
					Shares:           math.NewInt(20).ToLegacyDec(),
				},
				{
					DelegatorAddress: delAddr.String(),
					ValidatorAddress: valBAddr.String(),
					Shares:           math.NewInt(9).ToLegacyDec(),
				},
				{
					DelegatorAddress: delAddr.String(),
					ValidatorAddress: unbondedValAddr.String(),
					Shares:           math.NewInt(100).ToLegacyDec(),
				},
			}

			sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
			sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return(delegations, nil)
			sk.On("GetValidator", ctx, valAAddr).Return(valA, nil)
			sk.On("GetValidator", ctx, valBAddr).Return(valB, nil)
			sk.On("GetValidator", ctx, unbondedValAddr).Return(unbondedVal, nil)
			mockIterateDelegations(sk, ctx, delAddr, delegations)

			// The delegator has 20 + 9 bonded tokens across two validators.
			// The 100-token inactive delegation is ignored until it becomes
			// bonded. A 1-token delegate reaches exactly 30 / 100; 2 tokens
			// reaches 31 / 101 and must fail.
			tx := buildTx(t, &stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valBAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: tc.delegateAmt},
			})

			_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
				return ctx, nil
			})
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShareCapAuthz(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	validator := stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(30),
		DelegatorShares:   math.NewInt(30).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Shares:           math.NewInt(30).ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return(delegations, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	mockIterateDelegations(sk, ctx, delAddr, delegations)

	// Authz must not be a wrapper-based bypass. The inner delegate moves the
	// delegator from 30 / 100 to 31 / 101, which is above the cap.
	tx := buildTx(t, &authz.MsgExec{
		Grantee: sample.AccAddressBytes().String(),
		Msgs: []*codectypes.Any{
			mustAny(&stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
			}),
		},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapCancelUnbond(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	validator := stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(30),
		DelegatorShares:   math.NewInt(30).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAddr.String(),
			Shares:           math.NewInt(30).ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetValidator", ctx, valAddr).Return(validator, nil)
	mockIterateDelegations(sk, ctx, delAddr, delegations)

	// Canceling unbonding to a bonded validator adds active stake the same way
	// a delegate does. The final state is 31 / 101, so it must fail.
	tx := buildTx(t, &stakingtypes.MsgCancelUnbondingDelegation{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
		CreationHeight:   1,
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapRedelegate(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delAddr := sample.AccAddressBytes()
	srcValAddr := sdk.ValAddress(sample.AccAddressBytes())
	dstValAddr := sdk.ValAddress(sample.AccAddressBytes())
	srcVal := stakingtypes.Validator{
		OperatorAddress:   srcValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.OneInt(),
		DelegatorShares:   math.OneInt().ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	dstVal := stakingtypes.Validator{
		OperatorAddress:   dstValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(30),
		DelegatorShares:   math.NewInt(30).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: dstValAddr.String(),
			Shares:           math.NewInt(30).ToLegacyDec(),
		},
		{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: srcValAddr.String(),
			Shares:           math.OneInt().ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return(delegations, nil)
	sk.On("GetValidator", ctx, srcValAddr).Return(srcVal, nil)
	sk.On("GetValidator", ctx, dstValAddr).Return(dstVal, nil)
	sk.On("MaxValidators", ctx).Return(uint32(1), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{dstValAddr}}, nil)
	mockIterateDelegations(sk, ctx, delAddr, delegations)

	// Redelegating from an inactive validator to a bonded validator increases
	// active stake. The final state is 31 / 101, so it must fail.
	tx := buildTx(t, &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    delAddr.String(),
		ValidatorSrcAddress: srcValAddr.String(),
		ValidatorDstAddress: dstValAddr.String(),
		Amount:              sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapReplacement(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(200)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	attackerAddr := sample.AccAddressBytes()
	otherAddr := sample.AccAddressBytes()
	leavingValAddr := sdk.ValAddress(sample.AccAddressBytes())
	steadyValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	leavingVal := validator(leavingValAddr, stakingtypes.Bonded, math.NewInt(100))
	steadyVal := validator(steadyValAddr, stakingtypes.Bonded, math.NewInt(100))
	candidateVal := validator(candidateValAddr, stakingtypes.Unbonded, math.NewInt(96))
	candidateDelegations := []stakingtypes.Delegation{delegation(attackerAddr, candidateValAddr, math.NewInt(96))}
	leavingDelegations := []stakingtypes.Delegation{delegation(otherAddr, leavingValAddr, math.NewInt(100))}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, leavingVal)
	mockValidator(sk, ctx, steadyVal)
	mockValidator(sk, ctx, candidateVal)
	mockPowerStore(sk, ctx, 2, leavingValAddr, steadyValAddr, candidateValAddr)
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return(candidateDelegations, nil)
	sk.On("GetValidatorDelegations", ctx, leavingValAddr).Return(leavingDelegations, nil)
	mockIterateDelegations(sk, ctx, attackerAddr, candidateDelegations)

	// The tx only removes 5 active tokens, which is allowed by the 5% rule.
	// That drop lets a 96-token inactive candidate enter the active set, so the
	// candidate delegator's prospective 96 / 196 bonded share must be rejected.
	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: otherAddr.String(),
		ValidatorAddress: leavingValAddr.String(),
		Amount:           coin(5),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestFivePercentReplacement(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	otherAddr := sample.AccAddressBytes()
	leavingValAddr := valAddress(2)
	candidateValAddr := valAddress(1)
	leavingVal := validator(leavingValAddr, stakingtypes.Bonded, math.NewInt(100))
	candidateVal := validator(candidateValAddr, stakingtypes.Unbonded, math.NewInt(90))

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, leavingVal)
	mockValidator(sk, ctx, candidateVal)
	mockPowerStoreWithReduction(sk, ctx, 1, math.NewInt(10), leavingValAddr, candidateValAddr)

	// With a power reduction of 10, the 95-token outgoing validator and the
	// 90-token candidate tie at consensus power 9. The candidate address sorts
	// first, so the final bonded total is 90 and the full replacement exceeds
	// the allowed 5% decrease even though the explicit undelegate is only 5.
	tx := buildTx(t, &stakingtypes.MsgUndelegate{
		DelegatorAddress: otherAddr.String(),
		ValidatorAddress: leavingValAddr.String(),
		Amount:           coin(5),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorContains(t, err, "total stake decrease exceeds the allowed 5% threshold within a twelve-hour period")
}

func TestCreateValidatorInactive(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	newValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedVal := validator(bondedValAddr, stakingtypes.Bonded, currentTotal)

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, bondedVal)
	mockPowerStore(sk, ctx, 1, bondedValAddr)

	// Creating a validator records a candidate, but max validators is already
	// full and the existing bonded validator has more stake. The candidate stays
	// inactive, so neither the 5% rule nor the share cap should run on it.
	tx := buildTx(t, &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  newValAddr.String(),
		MinSelfDelegation: math.OneInt(),
		Value:             coin(10),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.NoError(t, err)
}

func TestCreateValidatorActive(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	newValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedVal := validator(bondedValAddr, stakingtypes.Bonded, currentTotal)

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	mockValidator(sk, ctx, bondedVal)
	mockPowerStore(sk, ctx, 2, bondedValAddr)

	// With room for a second active validator, the new 6-token validator would
	// enter the bonded set. That raises total bonded stake by more than 5%.
	tx := buildTx(t, &stakingtypes.MsgCreateValidator{
		ValidatorAddress:  newValAddr.String(),
		MinSelfDelegation: math.OneInt(),
		Value:             coin(6),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorContains(t, err, "total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
}

func TestShareCapDecimals(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(101)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Amount: currentTotal}))

	delAddr := sample.AccAddressBytes()
	valAAddr := sdk.ValAddress(sample.AccAddressBytes())
	valBAddr := sdk.ValAddress(sample.AccAddressBytes())
	valA := validator(valAAddr, stakingtypes.Bonded, math.NewInt(100))
	valA.DelegatorShares = math.NewInt(1000).ToLegacyDec()
	valB := validator(valBAddr, stakingtypes.Bonded, math.OneInt())
	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delAddr.String(),
			ValidatorAddress: valAAddr.String(),
			Shares:           math.NewInt(297).ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return(delegations, nil)
	mockValidator(sk, ctx, valA)
	mockValidator(sk, ctx, valB)
	mockIterateDelegations(sk, ctx, delAddr, delegations)

	// The existing delegation is worth 29.7 tokens, not 29. Truncating would
	// allow the new delegate, but the precise final share is 30.7 / 102.
	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valBAddr.String(),
		Amount:           coin(1),
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

type validatorPowerIterator struct {
	values [][]byte
	index  int
}

func (i *validatorPowerIterator) Domain() (start, end []byte) { return nil, nil }
func (i *validatorPowerIterator) Valid() bool                 { return i.index < len(i.values) }
func (i *validatorPowerIterator) Next()                       { i.index++ }
func (i *validatorPowerIterator) Key() []byte                 { return nil }
func (i *validatorPowerIterator) Value() []byte               { return i.values[i.index] }
func (i *validatorPowerIterator) Error() error                { return nil }
func (i *validatorPowerIterator) Close() error                { return nil }

func TestShareCapNewBonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delegatorAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	bondedDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: delegatorAddr.String(),
			ValidatorAddress: bondedValAddr.String(),
			Shares:           math.NewInt(29).ToLegacyDec(),
		},
	}
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.ZeroInt(),
		DelegatorShares:   math.LegacyZeroDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delegatorAddr).Return(bondedDelegations, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(2), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("IterateDelegatorDelegations", ctx, delegatorAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for _, delegation := range bondedDelegations {
			if fn(delegation) {
				return
			}
		}
	})

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	// The candidate starts unbonded, but this delegate would make it enter the
	// bonded set. The cap is checked against that prospective bonded state, so
	// the delegator's 31 / 102 final share is rejected.
	require.NoError(t, txBuilder.SetMsgs(&stakingtypes.MsgDelegate{
		DelegatorAddress: delegatorAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(2)},
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapExistingStake(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	attackerAddr := sample.AccAddressBytes()
	touchAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.NewInt(2),
		DelegatorShares:   math.NewInt(2).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	attackerDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: bondedValAddr.String(),
			Shares:           math.NewInt(29).ToLegacyDec(),
		},
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: candidateValAddr.String(),
			Shares:           math.NewInt(2).ToLegacyDec(),
		},
	}
	candidateDelegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: attackerAddr.String(),
			ValidatorAddress: candidateValAddr.String(),
			Shares:           math.NewInt(2).ToLegacyDec(),
		},
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, touchAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(2), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)
	sk.On("GetValidatorDelegations", ctx, candidateValAddr).Return(candidateDelegations, nil)
	mockIterateDelegations(sk, ctx, attackerAddr, attackerDelegations)
	mockIterateDelegations(sk, ctx, touchAddr, []stakingtypes.Delegation{})

	// The attacker already has 29 bonded tokens and 2 inactive tokens on the
	// candidate. A separate 1-token delegate makes the candidate bonded, so the
	// attacker's final active stake becomes 31 / 103 and must fail.
	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: touchAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorIs(t, err, types.ErrExceedsMaxStakeShare)
}

func TestShareCapStillUnbonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.NewInt(29),
		DelegatorShares:   math.NewInt(29).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(1), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)

	// The candidate grows from 29 to 30 tokens, but max validators is 1 and the
	// bonded validator still has 100. Since the candidate remains inactive,
	// the delegate does not increase bonded stake and should not trip the cap.
	tx := buildTx(t, &stakingtypes.MsgDelegate{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.NoError(t, err)
}

func TestFivePercentNewBonded(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	currentTotal := math.NewInt(100)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     currentTotal,
	}))

	delegatorAddr := sample.AccAddressBytes()
	bondedValAddr := sdk.ValAddress(sample.AccAddressBytes())
	bondedValidator := stakingtypes.Validator{
		OperatorAddress:   bondedValAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            currentTotal,
		DelegatorShares:   currentTotal.ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	candidateValAddr := sdk.ValAddress(sample.AccAddressBytes())
	candidateValidator := stakingtypes.Validator{
		OperatorAddress:   candidateValAddr.String(),
		Status:            stakingtypes.Unbonded,
		Tokens:            math.NewInt(5),
		DelegatorShares:   math.NewInt(5).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, delegatorAddr).Return([]stakingtypes.Delegation{}, nil)
	sk.On("GetValidator", ctx, candidateValAddr).Return(candidateValidator, nil)
	sk.On("GetValidator", ctx, bondedValAddr).Return(bondedValidator, nil)
	sk.On("MaxValidators", ctx).Return(uint32(2), nil)
	sk.On("PowerReduction", ctx).Return(math.OneInt())
	sk.On("ValidatorsPowerStoreIterator", ctx).Return(&validatorPowerIterator{values: [][]byte{bondedValAddr}}, nil)

	s := encoding.GetTestEncodingCfg()
	txBuilder := client.Context{}.WithTxConfig(s.TxConfig).TxConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(&stakingtypes.MsgDelegate{
		DelegatorAddress: delegatorAddr.String(),
		ValidatorAddress: candidateValAddr.String(),
		Amount:           sdk.Coin{Denom: "loya", Amount: math.OneInt()},
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})
	require.ErrorContains(t, err, "total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
}
