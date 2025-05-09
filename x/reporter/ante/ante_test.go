package ante

import (
	"errors"
	"fmt"
	"testing"

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
