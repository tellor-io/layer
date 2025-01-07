package ante

import (
	"errors"
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
	k, sk, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	decorator := NewTrackStakeChangesDecorator(k, sk)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil)
	err := k.Tracker.Set(ctx, types.StakeTracker{
		Expiration: nil,
		Amount:     math.NewInt(105),
	})
	require.NoError(t, err)
	testCases := []struct {
		name string
		msg  sdk.Msg
		err  error
	}{
		{
			name: "CreateValidator",
			msg: &stakingtypes.MsgCreateValidator{
				Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
		},
		{
			name: "CreateValidator",
			msg: &stakingtypes.MsgCreateValidator{
				Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(100)},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
		},
		{
			name: "Delegate",
			msg: &stakingtypes.MsgDelegate{
				DelegatorAddress: sample.AccAddressBytes().String(),
				ValidatorAddress: sample.AccAddressBytes().String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
		},
		{
			name: "BeginRedelegate",
			msg: &stakingtypes.MsgBeginRedelegate{
				DelegatorAddress:    sample.AccAddressBytes().String(),
				ValidatorSrcAddress: sample.AccAddressBytes().String(),
				ValidatorDstAddress: sample.AccAddressBytes().String(),
				Amount:              sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
			},
			err: nil,
		},
		{
			name: "CancelUnbondingDelegation",
			msg: &stakingtypes.MsgCancelUnbondingDelegation{
				DelegatorAddress: sample.AccAddressBytes().String(),
				ValidatorAddress: sample.AccAddressBytes().String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(100)},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
		},
		{
			name: "Undelegate",
			msg: &stakingtypes.MsgUndelegate{
				DelegatorAddress: sample.AccAddressBytes().String(),
				ValidatorAddress: sample.AccAddressBytes().String(),
				Amount:           sdk.Coin{Denom: "loya", Amount: math.NewInt(95)},
			},
			err: errors.New("total stake decrease exceeds the allowed 5% threshold within a twelve-hour period"),
		},
		{
			name: "Other message type",
			msg: &types.MsgUpdateParams{
				Authority: sample.AccAddressBytes().String(),
				Params:    types.Params{},
			},
			err: nil,
		},
		{
			name: "empty authz exec",
			msg:  &authz.MsgExec{},
			err:  nil,
		},
		{
			name: "authz exec with stake change > 5%",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&stakingtypes.MsgCreateValidator{
						Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1000)},
					}),
				},
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
		},
		{
			name: "authz exec with stake change < 5%",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&stakingtypes.MsgCreateValidator{
						Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
					}),
				},
			},
			err: nil,
		},
		{
			name: "authz exec wrapped in exec with stake change < 5%",
			msg: &authz.MsgExec{
				Grantee: sample.AccAddressBytes().String(),
				Msgs: []*codectypes.Any{
					mustAny(&authz.MsgExec{
						Grantee: sample.AccAddressBytes().String(),
						Msgs: []*codectypes.Any{
							mustAny(&stakingtypes.MsgCreateValidator{
								Value: sdk.Coin{Denom: "loya", Amount: math.NewInt(1)},
							}),
						},
					}),
				},
			},
			err: nil,
		},
		{
			name: "authz exec wrapped in exec with stake change > 5%",
			msg: &authz.MsgExec{
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
			},
			err: errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period"),
		},
	}
	s := encoding.GetTestEncodingCfg()
	clientCtx := client.Context{}.
		WithTxConfig(s.TxConfig)

	txBuilder := clientCtx.TxConfig.NewTxBuilder()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
