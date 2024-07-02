package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (k *KeeperTestSuite) TestExecuteVote() {
	dispute := k.dispute()

	// slash amount = 10000
	dispute.FeeTotal = math.NewInt(10000)
	feepayer1 := sample.AccAddressBytes()
	feepayer2 := sample.AccAddressBytes()
	feePayers := []types.PayerInfo{
		{Amount: math.NewInt(8000), FromBond: true},
		{Amount: math.NewInt(2000), FromBond: false},
	}
	vote := types.Vote{
		Id:         dispute.DisputeId,
		VoteEnd:    k.ctx.BlockTime(),
		VoteResult: types.VoteResult_NO_TALLY,
		Executed:   true,
	}

	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	k.Error(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId), "can't execute, dispute not resolved")

	dispute.DisputeEndTime = k.ctx.BlockTime()
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))
	k.Error(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId), "can't execute, dispute not resolved")

	vote.VoteResult = types.VoteResult_SUPPORT
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))
	k.Error(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId), "vote already executed")

	vote.Executed = false
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))

	k.ctx = k.ctx.WithBlockTime(k.ctx.BlockTime().Add(1))
	k.NoError(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId))

	k.NoError(k.disputeKeeper.DisputeFeePayer.Set(k.ctx, collections.Join(dispute.DisputeId, feepayer1.Bytes()), feePayers[0]))
	k.NoError(k.disputeKeeper.DisputeFeePayer.Set(k.ctx, collections.Join(dispute.DisputeId, feepayer2.Bytes()), feePayers[1]))
	msg := &types.MsgWithdrawFeeRefund{CallerAddress: sample.AccAddressBytes().String(), Id: dispute.DisputeId, PayerAddress: feepayer1.String()}
	k.reporterKeeper.On("FeeRefund", k.ctx, dispute.HashId, math.NewInt(8000)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(8000)))).Return(nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, math.NewInt(8000)).Return(nil)
	_, err := k.msgServer.WithdrawFeeRefund(k.ctx, msg)
	k.NoError(err)

	msg = &types.MsgWithdrawFeeRefund{CallerAddress: sample.AccAddressBytes().String(), Id: dispute.DisputeId, PayerAddress: feepayer2.String()}
	k.bankKeeper.On("SendCoinsFromModuleToAccount", k.ctx, types.ModuleName, feepayer2, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(2000)))).Return(nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, math.NewInt(2000)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(2000)))).Return(nil)
	_, err = k.msgServer.WithdrawFeeRefund(k.ctx, msg)
	k.NoError(err)
}

func (k *KeeperTestSuite) TestRefundDisputeFee() {
	disputeFeeMinusBurn := math.NewInt(950)
	feepayer1 := sample.AccAddressBytes()
	feepayer2 := sample.AccAddressBytes()
	feePayers := []types.PayerInfo{
		{Amount: math.NewInt(800), FromBond: true},
		{Amount: math.NewInt(200), FromBond: false},
	}

	k.reporterKeeper.On("FeeRefund", k.ctx, []byte("hash"), math.NewInt(760)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(760)))).Return(nil)
	dust, err := k.disputeKeeper.RefundDisputeFee(k.ctx, feepayer1, feePayers[0], math.NewInt(1000), disputeFeeMinusBurn, []byte("hash"))
	k.NoError(err)
	k.True(math.LegacyZeroDec().Equal(dust))

	k.bankKeeper.On("SendCoinsFromModuleToAccount", k.ctx, types.ModuleName, feepayer2, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(190)))).Return(nil)
	dust, err = k.disputeKeeper.RefundDisputeFee(k.ctx, feepayer2, feePayers[1], math.NewInt(1000), disputeFeeMinusBurn, []byte("hash"))
	k.NoError(err)
	k.True(math.LegacyZeroDec().Equal(dust))
}

func (k *KeeperTestSuite) TestRewardReporterBondToFeePayers() {
	reporterBond := math.NewInt(1000)
	feepayer1 := sample.AccAddressBytes()
	feepayer2 := sample.AccAddressBytes()
	feepayer3 := sample.AccAddressBytes()
	feePayers := []types.PayerInfo{
		{Amount: math.NewInt(800), FromBond: true},
		{Amount: math.NewInt(200), FromBond: true},
	}
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, math.NewInt(800)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(800)))).Return(nil)
	dust, err := k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer1, feePayers[0], reporterBond, reporterBond)
	k.NoError(err)
	k.True(math.LegacyZeroDec().Equal(dust))
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, math.NewInt(200)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(200)))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer2, feePayers[1], reporterBond, reporterBond)
	k.NoError(err)
	k.True(math.LegacyZeroDec().Equal(dust))

	feePayers = []types.PayerInfo{
		{Amount: math.NewInt(8), FromBond: true},
		{Amount: math.NewInt(2), FromBond: true},
		{Amount: math.NewInt(3), FromBond: true},
	}
	totalFeesPaid := math.NewInt(13)
	share := feePayers[0].Amount.ToLegacyDec().Quo(totalFeesPaid.ToLegacyDec()).Mul(reporterBond.ToLegacyDec())
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, share.TruncateInt()).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(615)))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer1, feePayers[0], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(share.Sub(share.TruncateDec()), dust)
	share = feePayers[1].Amount.ToLegacyDec().Quo(totalFeesPaid.ToLegacyDec()).Mul(reporterBond.ToLegacyDec())
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, share.TruncateInt()).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(153)))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer2, feePayers[1], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(share.Sub(share.TruncateDec()), dust)
	share = feePayers[2].Amount.ToLegacyDec().Quo(totalFeesPaid.ToLegacyDec()).Mul(reporterBond.ToLegacyDec())

	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer3, share.TruncateInt()).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(230)))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer3, feePayers[2], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(share.Sub(share.TruncateDec()), dust)
}

func (k *KeeperTestSuite) TestRewardVoters() {
	remaining, err := k.disputeKeeper.RewardVoters(k.ctx, []keeper.VoterInfo{{Voter: sample.AccAddressBytes(), Power: math.OneInt(), Share: math.ZeroInt()}}, math.ZeroInt(), math.ZeroInt())
	k.NoError(err)
	k.Equal(math.ZeroInt(), remaining)

	voters := []keeper.VoterInfo{
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(1), Share: math.ZeroInt()},
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(1), Share: math.ZeroInt()},
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(1), Share: math.ZeroInt()},
	}
	modulAddr := authtypes.NewModuleAddress(types.ModuleName)
	k.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(modulAddr, nil)
	k.bankKeeper.On(
		"InputOutputCoins", k.ctx,
		banktypes.Input{Address: modulAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(99)))},
		[]banktypes.Output{
			{Address: voters[0].Voter.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(33)))},
			{Address: voters[1].Voter.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(33)))},
			{Address: voters[2].Voter.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(33)))},
		}).Return(nil, nil)
	remaining, err = k.disputeKeeper.RewardVoters(k.ctx, voters, math.NewInt(100), math.NewInt(3))
	k.NoError(err)
	k.Equal(math.OneInt(), remaining)
}

func (k *KeeperTestSuite) TestCalculateVoterShare() {
	totalPower := math.NewInt(3)
	totalTokens := math.NewInt(100)
	voters := []keeper.VoterInfo{
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(1), Share: math.ZeroInt()},
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(1), Share: math.ZeroInt()},
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(1), Share: math.ZeroInt()},
	}
	voters, remainder := k.disputeKeeper.CalculateVoterShare(k.ctx, voters, totalTokens, totalPower)
	k.Equal(math.OneInt(), remainder)
	k.Equal(math.NewInt(33), voters[0].Share)
	k.Equal(math.NewInt(33), voters[1].Share)
	k.Equal(math.NewInt(33), voters[2].Share)

	totalPower = math.NewInt(5)
	totalTokens = math.NewInt(190)
	voters = []keeper.VoterInfo{
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(3), Share: math.ZeroInt()},
		{Voter: sample.AccAddressBytes(), Power: math.NewInt(2), Share: math.ZeroInt()},
	}

	voters, remainder = k.disputeKeeper.CalculateVoterShare(k.ctx, voters, totalTokens, totalPower)
	// 3/5 = 0.6 * 190 = 114
	k.Equal(math.ZeroInt(), remainder)
	k.Equal(math.NewInt(114), voters[0].Share)
	// 2/5 = 0.4 * 190 = 76
	k.Equal(math.NewInt(76), voters[1].Share)
}
