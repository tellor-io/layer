package keeper_test

import (
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *KeeperTestSuite) TestMsgServerAddFeeToDispute() {
	creator := sample.AccAddressBytes()
	msg := types.MsgAddFeeToDispute{
		Creator:     creator.String(),
		DisputeId:   1,
		Amount:      sdk.NewCoin("loya", math.NewInt(10000)),
		PayFromBond: false,
	}

	msg.Amount.Denom = "bond"
	res, err := k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "fee must be paid in loya")
	k.Nil(res)

	msg.Amount.Denom = "loya"
	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "not found")
	k.Nil(res)

	dispute := k.dispute()
	dispute.FeeTotal = dispute.SlashAmount
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	res, err = k.msgServer.AddFeeToDispute(k.ctx.WithBlockTime(time.Now()), &msg)
	k.ErrorContains(err, "dispute time expired")
	k.Nil(res)

	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "dispute fee already met")
	k.Nil(res)

	dispute.FeeTotal = math.ZeroInt()
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))
	fee := sdk.NewCoin(layer.BondDenom, math.NewInt(10000))
	k.oracleKeeper.On("FlagAggregateReport", k.ctx, mock.Anything).Return(nil)
	k.bankKeeper.On("HasBalance", k.ctx, creator, fee).Return(true)
	k.bankKeeper.On("SendCoinsFromAccountToModule", k.ctx, creator, types.ModuleName, sdk.NewCoins(fee)).Return(nil)
	k.reporterKeeper.On("EscrowReporterStake", k.ctx, sdk.MustAccAddressFromBech32(dispute.ReportEvidence.Reporter), dispute.ReportEvidence.Power, uint64(1), dispute.SlashAmount, dispute.HashId).Return(nil)
	// jail duration is 0
	k.reporterKeeper.On("JailReporter", k.ctx, sdk.MustAccAddressFromBech32(dispute.ReportEvidence.Reporter), uint64(0)).Return(nil)

	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.NoError(err)
	k.NotNil(res)
}
