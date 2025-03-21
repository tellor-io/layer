package keeper_test

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *KeeperTestSuite) TestMsgServerAddFeeToDispute() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
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

	dispute := k.dispute(k.ctx)
	dispute.FeeTotal = dispute.SlashAmount
	dispute.InitialEvidence.QueryId = []byte("query")
	fmt.Println("dispute: ", dispute)
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "dispute fee already met")
	k.Nil(res)

	dispute.FeeTotal = math.ZeroInt()
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))
	fee := sdk.NewCoin(layer.BondDenom, math.NewInt(10000))
	k.oracleKeeper.On("FlagAggregateReport", k.ctx, mock.Anything).Return(nil)
	k.bankKeeper.On("HasBalance", k.ctx, creator, fee).Return(true)
	k.bankKeeper.On("SendCoinsFromAccountToModule", k.ctx, creator, types.ModuleName, sdk.NewCoins(fee)).Return(nil)
	k.reporterKeeper.On("EscrowReporterStake", k.ctx, sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter), dispute.InitialEvidence.Power, uint64(1), dispute.SlashAmount, dispute.InitialEvidence.QueryId, dispute.HashId).Return(nil)
	// jail duration is 0
	k.reporterKeeper.On("JailReporter", k.ctx, sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter), uint64(0)).Return(nil)

	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.NoError(err)
	k.NotNil(res)

	// try to add again
	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "dispute fee already met")
	k.Nil(res)
}
