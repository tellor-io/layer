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

func mockDelegation(sk *mocks.StakingKeeper, ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress, tokens math.Int) {
	sk.On("GetDelegation", ctx, delegator, validator).Return(delegation(delegator, validator, tokens), nil)
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
				mockDelegation(sk, ctx, delAddr, srcValAddr, math.NewInt(100))
				mockPowerStore(sk, ctx, 1, srcValAddr)
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
			ctx = ctx.WithBlockHeight(1)
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

type validatorPowerIterator struct {
	values [][]byte
	index  int
}

func (i *validatorPowerIterator) Domain() (start, end []byte) { return nil, nil }
func (i *validatorPowerIterator) Valid() bool                 { return i.index < len(i.values) }
func (i *validatorPowerIterator) Next()                       { i.index++ }
func (i *validatorPowerIterator) Key() []byte                 { return nil }
func (i *validatorPowerIterator) Value() []byte               { return i.values[i.index] }
func (i *validatorPowerIterator) Error() error {
	if !i.Valid() {
		return fmt.Errorf("invalid cacheMergeIterator")
	}
	return nil
}
func (i *validatorPowerIterator) Close() error { return nil }
