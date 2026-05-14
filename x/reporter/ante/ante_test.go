package ante

import (
	"errors"
	"fmt"
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

func mustAny(msg sdk.Msg) *codectypes.Any {
	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}
	return any
}

func TestNewTrackStakeChangesDecorator(t *testing.T) {
	k, sk, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	sk.On("IterateDelegatorDelegations", ctx, mock.Anything, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil)
	err := k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     math.NewInt(105),
	})
	delAddr := sample.AccAddressBytes()
	valSrcAddr := sdk.ValAddress(sample.AccAddressBytes())
	valDstAddr := sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, err)
	testCases := []struct {
		name  string
		msg   sdk.Msg
		err   error
		setup func()
	}{
		{
			name: "CreateValidator",
			msg: &stakingtypes.MsgCreateValidator{
				Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
			setup: func() {
			},
		},
		{
			name: "CreateValidator",
			msg: &stakingtypes.MsgCreateValidator{
				Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(100)},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
			setup: func() {
			},
		},
		{
			name: "Delegate",
			msg: &stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valSrcAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Once()
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{}, nil).Once()
			},
		},
		{
			name: "Delegate. Already has 10 delegations",
			msg: &stakingtypes.MsgDelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valSrcAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: types.ErrExceedsMaxDelegations,
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Once()
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, nil).Once()
			},
		},
		{
			name: "BeginRedelegate",
			msg: &stakingtypes.MsgBeginRedelegate{
				DelegatorAddress:    delAddr.String(),
				ValidatorSrcAddress: valSrcAddr.String(),
				ValidatorDstAddress: valDstAddr.String(),
				Amount:              sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Twice()
				sk.On("GetValidator", ctx, valDstAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Twice()
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{}, nil).Once()
			},
		},
		{
			name: "BeginRedelegate. With 10 validators. Using Whole amount",
			msg: &stakingtypes.MsgBeginRedelegate{
				DelegatorAddress:    delAddr.String(),
				ValidatorSrcAddress: valSrcAddr.String(),
				ValidatorDstAddress: valDstAddr.String(),
				Amount:              sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Twice()
				sk.On("GetValidator", ctx, valDstAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Twice()
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{{ValidatorAddress: valSrcAddr.String(), Shares: math.LegacyNewDecFromInt(math.NewInt(1))}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, nil).Once()
			},
		},
		{
			name: "BeginRedelegate. With 10 validators. Using Not Whole amount",
			msg: &stakingtypes.MsgBeginRedelegate{
				DelegatorAddress:    delAddr.String(),
				ValidatorSrcAddress: valSrcAddr.String(),
				ValidatorDstAddress: valDstAddr.String(),
				Amount:              sdk.Coin{Denom: "loya", Amount: math.NewInt(100)},
			},
			err: types.ErrExceedsMaxDelegations,
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Twice()
				sk.On("GetValidator", ctx, valDstAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Twice()
				sk.On("GetAllDelegatorDelegations", ctx, delAddr).Return([]stakingtypes.Delegation{{ValidatorAddress: valSrcAddr.String(), Shares: math.LegacyNewDecFromInt(math.NewInt(1))}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, nil).Once()
			},
		},
		{
			name: "CancelUnbondingDelegation",
			msg: &stakingtypes.MsgCancelUnbondingDelegation{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valSrcAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(100)},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Once()
			},
		},
		{
			name: "Undelegate",
			msg: &stakingtypes.MsgUndelegate{
				DelegatorAddress: delAddr.String(),
				ValidatorAddress: valSrcAddr.String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(95)},
			},
			err: errors.New("total stake decrease exceeds the allowed 5% threshold within a twelve-hour period"),
			setup: func() {
				sk.On("GetValidator", ctx, valSrcAddr).Return(stakingtypes.Validator{Status: stakingtypes.Bonded}, nil).Once()
			},
		},
		{
			name: "Other message type",
			msg: &types.MsgUpdateParams{
				Authority: sample.AccAddressBytes().String(),
				Params:    types.Params{},
			},
			err: nil,
			setup: func() {
			},
		},
		{
			name: "empty authz exec",
			msg:  &authz.MsgExec{},
			err:  nil,
			setup: func() {
			},
		},
		{
			name: "stake change > 5% wrapped once",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&stakingtypes.MsgCreateValidator{
						Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1000)},
					}),
				},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
			setup: func() {
			},
		},
		{
			name: "stake change < 5% wrapped once",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&stakingtypes.MsgCreateValidator{
						Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
					}),
				},
			},
			err: nil,
			setup: func() {
			},
		},
		{
			name: "stake change < 5% wrapped twice",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&authz.MsgExec{
						Grantee: sample.AccAddressBytes().String(),
						Msgs: []*codectypes.Any{
							mustAny(&authz.MsgExec{
								Grantee: sample.AccAddressBytes().String(),
								Msgs: []*codectypes.Any{
									mustAny(&authz.MsgExec{
										Grantee: sample.AccAddressBytes().String(),
										Msgs: []*codectypes.Any{
											mustAny(&authz.MsgExec{
												Grantee: sample.AccAddressBytes().String(),
												Msgs: []*codectypes.Any{
													mustAny(&authz.MsgExec{
														Grantee: sample.AccAddressBytes().String(),
														Msgs: []*codectypes.Any{
															mustAny(&authz.MsgExec{
																Grantee: sample.AccAddressBytes().String(),
																Msgs: []*codectypes.Any{
																	mustAny(&stakingtypes.MsgCreateValidator{
																		Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1000)},
																	}),
																},
															}),
														},
													}),
												},
											}),
										},
									}),
								},
							}),
						},
					}),
				},
			},
			err: fmt.Errorf("nested message count exceeds the maximum allowed: Limit is %d", MaxNestedMsgCount),
			setup: func() {
			},
		},
		{
			name: "stake change > 5% wrapped twice",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&authz.MsgExec{
						Grantee: sample.AccAddressBytes().String(),
						Msgs: []*codectypes.Any{
							mustAny(&authz.MsgExec{
								Grantee: sample.AccAddressBytes().String(),
								Msgs: []*codectypes.Any{
									mustAny(&stakingtypes.MsgCreateValidator{
										Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1000)},
									}),
								},
							}),
						},
					}),
				},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
			setup: func() {
			},
		},
		{
			name: "nested message count exceeds the maximum allowed",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&authz.MsgExec{
						Grantee: sample.AccAddressBytes().String(),
						Msgs: []*codectypes.Any{
							mustAny(&authz.MsgExec{
								Grantee: sample.AccAddressBytes().String(),
								Msgs: []*codectypes.Any{
									mustAny(&authz.MsgExec{
										Grantee: sample.AccAddressBytes().String(),
										Msgs: []*codectypes.Any{
											mustAny(&authz.MsgExec{
												Grantee: sample.AccAddressBytes().String(),
												Msgs: []*codectypes.Any{
													mustAny(&authz.MsgExec{
														Grantee: sample.AccAddressBytes().String(),
														Msgs: []*codectypes.Any{
															mustAny(&authz.MsgExec{
																Grantee: sample.AccAddressBytes().String(),
																Msgs: []*codectypes.Any{
																	mustAny(&stakingtypes.MsgCreateValidator{
																		Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1000)},
																	}),
																},
															}),
														},
													}),
												},
											}),
										},
									}),
								},
							}),
						},
					}),
				},
			},
			err: errors.New("nested message count exceeds the maximum allowed: Limit is 7"),
			setup: func() {
			},
		},
	}

	s := encoding.GetTestEncodingCfg()
	clientCtx := client.Context{}.
		WithTxConfig(s.TxConfig)

	txBuilder := clientCtx.TxConfig.NewTxBuilder()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := txBuilder.SetMsgs(tc.msg)
			require.NoError(t, err)
			tx := txBuilder.GetTx()
			_, err = decorator.AnteHandle(ctx, tx, false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
				return ctx, nil
			})

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStakeShare(t *testing.T) {
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

func TestStakeShareSameTxDrop(t *testing.T) {
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

func TestStakeShareReduction(t *testing.T) {
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

func TestStakeShareFinalTotal(t *testing.T) {
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
		Tokens:            math.NewInt(2),
		DelegatorShares:   math.NewInt(2).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}

	sk.On("TotalBondedTokens", ctx).Return(currentTotal, nil)
	sk.On("GetAllDelegatorDelegations", ctx, aliceAddr).Return(aliceDelegations, nil)
	sk.On("GetValidator", ctx, aliceValAddr).Return(aliceValidator, nil)
	sk.On("GetValidator", ctx, bobValAddr).Return(bobValidator, nil)
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

func TestUnbondedBondingStakeShare(t *testing.T) {
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

func TestTrackStakeChangesDecoratorBlocksUnbondedValidatorBondingOverFivePercent(t *testing.T) {
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
