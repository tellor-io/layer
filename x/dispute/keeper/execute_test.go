package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *KeeperTestSuite) TestExecuteVote() {
	dispute := k.dispute()

	// slash amount = 10000
	dispute.FeeTotal = math.NewInt(10000)
	dispute.DisputeFee = dispute.FeeTotal
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
	voteCounts := types.StakeholderVoteCounts{
		Users:     types.VoteCounts{Support: 1, Against: 0, Invalid: 0},
		Reporters: types.VoteCounts{Support: 1, Against: 0, Invalid: 0},
		// Tokenholders: types.VoteCounts{Support: 1, Against: 0, Invalid: 0},
		Team: types.VoteCounts{Support: 1, Against: 0, Invalid: 0},
	}
	k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, dispute.DisputeId, voteCounts))

	// vote and dispute set, dispute status not resolved
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))
	k.Error(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId), "can't execute, dispute not resolved")

	// dispute time ended but vote result not
	dispute.DisputeEndTime = k.ctx.BlockTime()
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))
	k.Error(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId), "can't execute, dispute not resolved")

	// vote aleady executed
	vote.VoteResult = types.VoteResult_SUPPORT
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))
	k.Error(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId), "vote already executed")

	// actually execute vote
	vote.Executed = false
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))
	k.ctx = k.ctx.WithBlockTime(k.ctx.BlockTime().Add(1))
	k.bankKeeper.On("BurnCoins", k.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("loya", dispute.BurnAmount.QuoRaw(2)))).Return(nil)
	k.NoError(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId))

	// withdraw fee refund for feepayer1
	k.NoError(k.disputeKeeper.DisputeFeePayer.Set(k.ctx, collections.Join(dispute.DisputeId, feepayer1.Bytes()), feePayers[0]))
	k.NoError(k.disputeKeeper.DisputeFeePayer.Set(k.ctx, collections.Join(dispute.DisputeId, feepayer2.Bytes()), feePayers[1]))
	msg := &types.MsgWithdrawFeeRefund{CallerAddress: sample.AccAddressBytes().String(), Id: dispute.DisputeId, PayerAddress: feepayer1.String()}
	k.reporterKeeper.On("FeeRefund", k.ctx, dispute.HashId, math.NewInt(7600)).Return(nil).Once()
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(7600)))).Return(nil).Once()
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, math.NewInt(8000)).Return(nil).Once()
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(8000)))).Return(nil).Once()
	_, err := k.msgServer.WithdrawFeeRefund(k.ctx, msg)
	k.NoError(err)

	// wqithdraw fee refund for feepayer2
	msg = &types.MsgWithdrawFeeRefund{CallerAddress: sample.AccAddressBytes().String(), Id: dispute.DisputeId, PayerAddress: feepayer2.String()}
	k.bankKeeper.On("SendCoinsFromModuleToAccount", k.ctx, types.ModuleName, feepayer2, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(1900)))).Return(nil).Once()
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, math.NewInt(2000)).Return(nil).Once()
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(2000)))).Return(nil).Once()
	_, err = k.msgServer.WithdrawFeeRefund(k.ctx, msg)
	k.NoError(err)
}

func (k *KeeperTestSuite) TestRefundDisputeFee() {
	// disputeFeeMinusBurn := math.NewInt(950)
	feepayer1 := sample.AccAddressBytes()
	feepayer2 := sample.AccAddressBytes()
	feePayers := []types.PayerInfo{
		{Amount: math.NewInt(800), FromBond: true},
		{Amount: math.NewInt(200), FromBond: false},
	}

	k.reporterKeeper.On("FeeRefund", k.ctx, []byte("hash"), math.NewInt(760)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(760)))).Return(nil)
	dust, err := k.disputeKeeper.RefundDisputeFee(k.ctx, feepayer1, feePayers[0], math.NewInt(1000), []byte("hash"), math.NewInt(1000))
	k.NoError(err)
	k.True(math.ZeroInt().Equal(dust))

	k.bankKeeper.On("SendCoinsFromModuleToAccount", k.ctx, types.ModuleName, feepayer2, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(190)))).Return(nil)
	dust, err = k.disputeKeeper.RefundDisputeFee(k.ctx, feepayer2, feePayers[1], math.NewInt(1000), []byte("hash"), math.NewInt(1000))
	k.NoError(err)
	k.True(math.ZeroInt().Equal(dust))
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
	k.True(math.ZeroInt().Equal(dust))
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, math.NewInt(200)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(200)))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer2, feePayers[1], reporterBond, reporterBond)
	k.NoError(err)
	k.True(math.ZeroInt().Equal(dust))

	feePayers = []types.PayerInfo{
		{Amount: math.NewInt(8), FromBond: true},
		{Amount: math.NewInt(2), FromBond: true},
		{Amount: math.NewInt(3), FromBond: true},
	}
	totalFeesPaid := math.NewInt(13)

	shareFixed12 := feePayers[0].Amount.Mul(reporterBond).Mul(layertypes.PowerReduction).Quo(totalFeesPaid)
	shareFixed6 := shareFixed12.Quo(layertypes.PowerReduction)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, shareFixed6).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", shareFixed6))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer1, feePayers[0], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(shareFixed12.Mod(layertypes.PowerReduction), dust)

	shareFixed12 = feePayers[1].Amount.Mul(reporterBond).Mul(layertypes.PowerReduction).Quo(totalFeesPaid)
	shareFixed6 = shareFixed12.Quo(layertypes.PowerReduction)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, shareFixed6).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", shareFixed6))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer2, feePayers[1], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(shareFixed12.Mod(layertypes.PowerReduction), dust)

	shareFixed12 = feePayers[2].Amount.Mul(reporterBond).Mul(layertypes.PowerReduction).Quo(totalFeesPaid)
	shareFixed6 = shareFixed12.Quo(layertypes.PowerReduction)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer3, shareFixed6).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", shareFixed6))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer3, feePayers[2], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(shareFixed12.Mod(layertypes.PowerReduction), dust)
}

func (k *KeeperTestSuite) TestGetSumOfAllGroupVotesAllRounds() {
	dispute := k.dispute()
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	// set vote counts for current dispute
	currentVoteCounts := types.StakeholderVoteCounts{
		Users:     types.VoteCounts{Support: 10, Against: 5, Invalid: 2}, // 17
		Reporters: types.VoteCounts{Support: 8, Against: 3, Invalid: 1},  // 12
		// Tokenholders: types.VoteCounts{Support: 15, Against: 7, Invalid: 3}, // 25
		Team: types.VoteCounts{Support: 5, Against: 2, Invalid: 1}, // 8 total=37
	}
	k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, dispute.DisputeId, currentVoteCounts))

	// test no previous disputes
	expectedTotalSum := math.NewInt(37)
	totalSum, err := k.disputeKeeper.GetSumOfAllGroupVotesAllRounds(k.ctx, dispute.DisputeId)
	k.NoError(err)
	k.True(expectedTotalSum.Equal(totalSum))

	// test with 3 previous dispute rounds
	prevDisputeIds := []uint64{2, 3, 4}
	prevVoteCounts := []types.StakeholderVoteCounts{
		{
			Users:     types.VoteCounts{Support: 5, Against: 3, Invalid: 1}, // 9
			Reporters: types.VoteCounts{Support: 4, Against: 2, Invalid: 0}, // 6
			// Tokenholders: types.VoteCounts{Support: 8, Against: 4, Invalid: 2}, // 14
			Team: types.VoteCounts{Support: 3, Against: 1, Invalid: 0}, // 4 total=19
		},
		{
			Users:     types.VoteCounts{Support: 7, Against: 4, Invalid: 2}, // 13
			Reporters: types.VoteCounts{Support: 6, Against: 3, Invalid: 1}, // 10
			// Tokenholders: types.VoteCounts{Support: 10, Against: 5, Invalid: 2}, // 17
			Team: types.VoteCounts{Support: 4, Against: 2, Invalid: 1}, // 7 total=30
		},
		{
			Users:     types.VoteCounts{Support: 3, Against: 2, Invalid: 0}, // 5
			Reporters: types.VoteCounts{Support: 2, Against: 1, Invalid: 0}, // 3
			// Tokenholders: types.VoteCounts{Support: 5, Against: 3, Invalid: 1}, // 9
			Team: types.VoteCounts{Support: 2, Against: 1, Invalid: 0}, // 3 total=11
		},
	}

	dispute.PrevDisputeIds = prevDisputeIds
	for i, id := range prevDisputeIds {
		k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, id, prevVoteCounts[i]))
	}

	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	// Calculate the expected total sum
	expectedTotalSum = math.NewInt(0).
		Add(math.NewInt(int64(17 + 12 + 8))). // Current dispute
		Add(math.NewInt(int64(9 + 6 + 4))).   // Previous dispute 1
		Add(math.NewInt(int64(13 + 10 + 7))). // Previous dispute 2
		Add(math.NewInt(int64(5 + 3 + 3)))    // Previous dispute 3

	// Call the function and check the result
	totalSum, err = k.disputeKeeper.GetSumOfAllGroupVotesAllRounds(k.ctx, dispute.DisputeId)
	k.NoError(err)
	k.True(expectedTotalSum.Equal(totalSum))
}
