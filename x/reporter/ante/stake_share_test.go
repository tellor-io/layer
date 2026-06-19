package ante

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/encoding"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

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
	mockDelegation(sk, ctx, otherAddr, otherValAddr, math.NewInt(10))
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
	mockDelegation(sk, ctx, delAddr, valAddr, math.NewInt(40))
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
	mockDelegation(sk, ctx, bobAddr, bobValAddr, math.NewInt(10))
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
	mockDelegation(sk, ctx, delAddr, srcValAddr, math.OneInt())
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
