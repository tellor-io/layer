package ante

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestMixedStakeAcquisitionPaths(t *testing.T) {
	directMessages := []struct {
		name string
		msg  sdk.Msg
	}{
		{
			name: "withdraw tip",
			msg: &types.MsgWithdrawTip{
				SelectorAddress:  sample.AccAddressBytes().String(),
				ValidatorAddress: testValAddress().String(),
			},
		},
		{
			name: "withdraw fee refund",
			msg: &disputetypes.MsgWithdrawFeeRefund{
				CallerAddress: sample.AccAddressBytes().String(),
				PayerAddress:  sample.AccAddressBytes().String(),
				Id:            1,
			},
		},
	}
	reporterAttributionMessages := []struct {
		name string
		msg  sdk.Msg
	}{
		{
			name: "create reporter",
			msg: &types.MsgCreateReporter{
				ReporterAddress:   sample.AccAddressBytes().String(),
				CommissionRate:    types.DefaultMinCommissionRate,
				MinTokensRequired: types.DefaultMinLoya,
				Moniker:           "reporter",
			},
		},
		{
			name: "select reporter",
			msg: &types.MsgSelectReporter{
				SelectorAddress: sample.AccAddressBytes().String(),
				ReporterAddress: sample.AccAddressBytes().String(),
			},
		},
		{
			name: "switch reporter",
			msg: &types.MsgSwitchReporter{
				SelectorAddress: sample.AccAddressBytes().String(),
				ReporterAddress: sample.AccAddressBytes().String(),
			},
		},
	}

	for _, direct := range directMessages {
		t.Run(direct.name+" rejects acquisition before", func(t *testing.T) {
			_, _, ctx, decorator := setupPowerCapDecorator(t, 100)
			tx := buildTx(t, createValidatorMsg(), direct.msg)
			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			require.ErrorIs(t, err, types.ErrMixedStakeAcquisitionPaths)
		})

		t.Run(direct.name+" rejects nested acquisition after", func(t *testing.T) {
			_, _, ctx, decorator := setupPowerCapDecorator(t, 100)
			tx := buildTx(t, direct.msg, nestedExec(1, createValidatorMsg()))
			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			require.ErrorIs(t, err, types.ErrMixedStakeAcquisitionPaths)
		})

		t.Run(direct.name+" allows non-acquisition staking", func(t *testing.T) {
			_, _, ctx, decorator := setupPowerCapDecorator(t, 100)
			tx := buildTx(t, direct.msg, &stakingtypes.MsgEditValidator{
				ValidatorAddress: testValAddress().String(),
			})
			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			require.NoError(t, err)
		})

		t.Run(direct.name+" allows unrelated message", func(t *testing.T) {
			_, _, ctx, decorator := setupPowerCapDecorator(t, 100)
			tx := buildTx(t, direct.msg, &banktypes.MsgSend{
				FromAddress: sample.AccAddressBytes().String(),
				ToAddress:   sample.AccAddressBytes().String(),
			})
			_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
			require.NoError(t, err)
		})

		for _, attribution := range reporterAttributionMessages {
			t.Run(direct.name+" rejects "+attribution.name+" before", func(t *testing.T) {
				_, _, ctx, decorator := setupPowerCapDecorator(t, 100)
				tx := buildTx(t, attribution.msg, direct.msg)
				_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
				require.ErrorIs(t, err, types.ErrMixedStakeAcquisitionPaths)
			})

			t.Run(direct.name+" rejects nested "+attribution.name+" after", func(t *testing.T) {
				_, _, ctx, decorator := setupPowerCapDecorator(t, 100)
				tx := buildTx(t, direct.msg, nestedExec(1, attribution.msg))
				_, err := decorator.AnteHandle(ctx, tx, false, noopNext)
				require.ErrorIs(t, err, types.ErrMixedStakeAcquisitionPaths)
			})
		}
	}
}

func createValidatorMsg() *stakingtypes.MsgCreateValidator {
	return &stakingtypes.MsgCreateValidator{
		ValidatorAddress: testValAddress().String(),
		Value:            coin(1),
	}
}

func testValAddress() sdk.ValAddress {
	return sdk.ValAddress(sample.AccAddressBytes())
}

func TestStakeAcquisitionMessageClassification(t *testing.T) {
	tests := []struct {
		name      string
		msg       sdk.Msg
		direct    bool
		projected bool
	}{
		{name: "withdraw tip", msg: &types.MsgWithdrawTip{}, direct: true},
		{name: "withdraw fee refund", msg: &disputetypes.MsgWithdrawFeeRefund{}, direct: true},
		{name: "create reporter", msg: &types.MsgCreateReporter{}, projected: true},
		{name: "select reporter", msg: &types.MsgSelectReporter{}, projected: true},
		{name: "switch reporter", msg: &types.MsgSwitchReporter{}, projected: true},
		{name: "create validator", msg: &stakingtypes.MsgCreateValidator{}, projected: true},
		{name: "delegate", msg: &stakingtypes.MsgDelegate{}, projected: true},
		{name: "redelegate", msg: &stakingtypes.MsgBeginRedelegate{}, projected: true},
		{name: "cancel unbonding", msg: &stakingtypes.MsgCancelUnbondingDelegation{}, projected: true},
		{name: "undelegate", msg: &stakingtypes.MsgUndelegate{}},
		{name: "unjail", msg: &slashingtypes.MsgUnjail{}},
		{name: "edit validator", msg: &stakingtypes.MsgEditValidator{}},
		{name: "bank send", msg: &banktypes.MsgSend{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.direct, isDirectStakeAcquisitionMessage(test.msg))
			require.Equal(t, test.projected, isProjectedPowerAcquisitionMessage(test.msg))
		})
	}
}
