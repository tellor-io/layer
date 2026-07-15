package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *KeeperTestSuite) TestExecuteVote() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)

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
	k.reporterKeeper.On("FeePaidFromStakeTotalByPayer", k.ctx, dispute.HashId, feepayer1).Return(math.NewInt(8000), nil).Once()
	k.reporterKeeper.On("FeeRefund", k.ctx, dispute.HashId, feepayer1, math.NewInt(7600)).Return(math.NewInt(7600), math.ZeroInt(), nil).Once()
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(7600)))).Return(nil).Once()
	k.reporterKeeper.On("DisputedDelegationTotal", k.ctx, dispute.HashId).Return(dispute.SlashAmount, nil).Once()
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, math.NewInt(8000)).Return(math.NewInt(8000), math.ZeroInt(), nil).Once()
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(8000)))).Return(nil).Once()
	_, err := k.msgServer.WithdrawFeeRefund(k.ctx, msg)
	k.NoError(err)

	// wqithdraw fee refund for feepayer2
	msg = &types.MsgWithdrawFeeRefund{CallerAddress: sample.AccAddressBytes().String(), Id: dispute.DisputeId, PayerAddress: feepayer2.String()}
	k.bankKeeper.On("SendCoinsFromModuleToAccount", k.ctx, types.ModuleName, feepayer2, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(1900)))).Return(nil).Once()
	k.reporterKeeper.On("DisputedDelegationTotal", k.ctx, dispute.HashId).Return(dispute.SlashAmount, nil).Once()
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, math.NewInt(2000)).Return(math.NewInt(2000), math.ZeroInt(), nil).Once()
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

	k.reporterKeeper.On("FeePaidFromStakeTotalByPayer", k.ctx, []byte("hash"), feepayer1).Return(math.NewInt(800), nil)
	k.reporterKeeper.On("FeeRefund", k.ctx, []byte("hash"), feepayer1, math.NewInt(760)).Return(math.NewInt(760), math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(760)))).Return(nil)
	dust, err := k.disputeKeeper.RefundDisputeFee(k.ctx, feepayer1, feePayers[0], math.NewInt(1000), []byte("hash"))
	k.NoError(err)
	k.True(math.ZeroInt().Equal(dust))

	shortPayer := sample.AccAddressBytes()
	k.reporterKeeper.On("FeePaidFromStakeTotalByPayer", k.ctx, []byte("short-hash"), shortPayer).Return(math.NewInt(799), nil)
	k.reporterKeeper.On("FeeRefund", k.ctx, []byte("short-hash"), shortPayer, math.NewInt(759)).Return(math.NewInt(759), math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(759)))).Return(nil)
	dust, err = k.disputeKeeper.RefundDisputeFee(k.ctx, shortPayer, feePayers[0], math.NewInt(1000), []byte("short-hash"))
	k.NoError(err)
	k.True(math.ZeroInt().Equal(dust))

	k.bankKeeper.On("SendCoinsFromModuleToAccount", k.ctx, types.ModuleName, feepayer2, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(190)))).Return(nil)
	dust, err = k.disputeKeeper.RefundDisputeFee(k.ctx, feepayer2, feePayers[1], math.NewInt(1000), []byte("hash"))
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
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, math.NewInt(800)).Return(math.NewInt(800), math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(800)))).Return(nil)
	dust, err := k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer1, feePayers[0], reporterBond, reporterBond)
	k.NoError(err)
	k.True(math.ZeroInt().Equal(dust))
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, math.NewInt(200)).Return(math.NewInt(200), math.ZeroInt(), nil)
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
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer1, shareFixed6).Return(shareFixed6, math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", shareFixed6))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer1, feePayers[0], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(shareFixed12.Mod(layertypes.PowerReduction), dust)

	shareFixed12 = feePayers[1].Amount.Mul(reporterBond).Mul(layertypes.PowerReduction).Quo(totalFeesPaid)
	shareFixed6 = shareFixed12.Quo(layertypes.PowerReduction)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer2, shareFixed6).Return(shareFixed6, math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", shareFixed6))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer2, feePayers[1], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(shareFixed12.Mod(layertypes.PowerReduction), dust)

	shareFixed12 = feePayers[2].Amount.Mul(reporterBond).Mul(layertypes.PowerReduction).Quo(totalFeesPaid)
	shareFixed6 = shareFixed12.Quo(layertypes.PowerReduction)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feepayer3, shareFixed6).Return(shareFixed6, math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", shareFixed6))).Return(nil)
	dust, err = k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feepayer3, feePayers[2], totalFeesPaid, reporterBond)
	k.NoError(err)
	k.Equal(shareFixed12.Mod(layertypes.PowerReduction), dust)
}

func (k *KeeperTestSuite) TestGetSumOfUserAndReporterVotesAllRounds() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	// set vote counts for current dispute; team votes are set but must not be counted
	currentVoteCounts := types.StakeholderVoteCounts{
		Users:     types.VoteCounts{Support: 10, Against: 5, Invalid: 2}, // 17
		Reporters: types.VoteCounts{Support: 8, Against: 3, Invalid: 1},  // 12
		Team:      types.VoteCounts{Support: 5, Against: 2, Invalid: 1},  // excluded, total=29
	}
	k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, dispute.DisputeId, currentVoteCounts))

	// test no previous disputes
	expectedTotalSum := math.NewInt(29)
	totalSum, err := k.disputeKeeper.GetSumOfUserAndReporterVotesAllRounds(k.ctx, dispute.DisputeId)
	k.NoError(err)
	k.True(expectedTotalSum.Equal(totalSum))

	// test with 3 previous dispute rounds
	prevDisputeIds := []uint64{2, 3, 4}
	prevVoteCounts := []types.StakeholderVoteCounts{
		{
			Users:     types.VoteCounts{Support: 5, Against: 3, Invalid: 1}, // 9
			Reporters: types.VoteCounts{Support: 4, Against: 2, Invalid: 0}, // 6
			Team:      types.VoteCounts{Support: 3, Against: 1, Invalid: 0}, // excluded, total=15
		},
		{
			Users:     types.VoteCounts{Support: 7, Against: 4, Invalid: 2}, // 13
			Reporters: types.VoteCounts{Support: 6, Against: 3, Invalid: 1}, // 10
			Team:      types.VoteCounts{Support: 4, Against: 2, Invalid: 1}, // excluded, total=23
		},
		{
			Users:     types.VoteCounts{Support: 3, Against: 2, Invalid: 0}, // 5
			Reporters: types.VoteCounts{Support: 2, Against: 1, Invalid: 0}, // 3
			Team:      types.VoteCounts{Support: 2, Against: 1, Invalid: 0}, // excluded, total=8
		},
	}

	dispute.PrevDisputeIds = prevDisputeIds
	for i, id := range prevDisputeIds {
		k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, id, prevVoteCounts[i]))
	}

	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	// Calculate the expected total sum (team votes excluded)
	expectedTotalSum = math.NewInt(0).
		Add(math.NewInt(int64(17 + 12))). // Current dispute
		Add(math.NewInt(int64(9 + 6))).   // Previous dispute 1
		Add(math.NewInt(int64(13 + 10))). // Previous dispute 2
		Add(math.NewInt(int64(5 + 3)))    // Previous dispute 3

	// Call the function and check the result
	totalSum, err = k.disputeKeeper.GetSumOfUserAndReporterVotesAllRounds(k.ctx, dispute.DisputeId)
	k.NoError(err)
	k.True(expectedTotalSum.Equal(totalSum))
}

// TestGetSumOfUserAndReporterVotesAllRoundsDedup guards that the same round id is
// counted once even when PrevDisputeIds contains the current dispute id. This is
// the normal shape produced by AddDisputeRound.
func (k *KeeperTestSuite) TestGetSumOfUserAndReporterVotesAllRoundsDedup() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)
	dispute.DisputeId = 10
	dispute.PrevDisputeIds = []uint64{10, 20, 30} // current id 10 appears in the list
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, 10, types.StakeholderVoteCounts{
		Users: types.VoteCounts{Support: 5, Against: 3, Invalid: 2}, // 10
	}))
	k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, 20, types.StakeholderVoteCounts{
		Reporters: types.VoteCounts{Support: 4}, // 4
	}))
	k.NoError(k.disputeKeeper.VoteCountsByGroup.Set(k.ctx, 30, types.StakeholderVoteCounts{
		Users: types.VoteCounts{Against: 1}, // 1
	}))

	totalSum, err := k.disputeKeeper.GetSumOfUserAndReporterVotesAllRounds(k.ctx, dispute.DisputeId)
	k.NoError(err)
	// 10 + 4 + 1 = 15; round 10 counted once despite appearing as current + in PrevDisputeIds
	k.True(math.NewInt(15).Equal(totalSum))
}

// TestExecuteVoteAgainst guards that the AGAINST branch passes the fee upside as
// extraReturn without mutating Dispute.SlashAmount. Previously the code added the
// fee upside to SlashAmount, relied on the reporter keeper to infer it from
// amt > snapshot.Total, and restored it at the end; that left an early-return
// bug window and could spend dispute-fee funds to cover phantom slash principal.
func (k *KeeperTestSuite) TestExecuteVoteAgainst() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)
	dispute.DisputeFee = math.NewInt(10000)
	dispute.DisputeEndTime = k.ctx.BlockTime().Add(-1 * time.Hour)
	dispute.DisputeStatus = types.Resolved
	dispute.Open = false
	dispute.BurnAmount = math.NewInt(500)
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	vote := types.Vote{
		Id:         dispute.DisputeId,
		VoteEnd:    k.ctx.BlockTime(),
		VoteResult: types.VoteResult_AGAINST,
		Executed:   false,
	}
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, dispute.DisputeId, vote))

	// No vote-count entry -> totalVoterPower == 0, so halfBurnAmount = BurnAmount.
	k.bankKeeper.On("BurnCoins", k.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("loya", dispute.BurnAmount))).Return(nil)

	fivePercent := dispute.DisputeFee.ToLegacyDec().Quo(math.LegacyNewDec(20)).TruncateInt()
	extraReturn := dispute.DisputeFee.Sub(fivePercent)
	bondedReturn := dispute.SlashAmount.Add(extraReturn)
	reporterAddr, err := sdk.AccAddressFromBech32(dispute.InitialEvidence.GetReporter())
	k.NoError(err)

	k.reporterKeeper.On("ReturnSlashedTokens", k.ctx, dispute.HashId, extraReturn).Return(bondedReturn, math.ZeroInt(), nil).Once()
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", bondedReturn))).Return(nil).Once()
	k.reporterKeeper.On("UpdateJailedUntilOnFailedDispute", k.ctx, reporterAddr, dispute.InitialEvidence.BlockNumber, dispute.HashId).Return(nil).Once()

	k.NoError(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId))

	stored, err := k.disputeKeeper.Disputes.Get(k.ctx, dispute.DisputeId)
	k.NoError(err)
	k.Equal(math.NewInt(10000), stored.SlashAmount, "SlashAmount must remain the escrowed principal, not include the fee upside")
	k.Equal(math.ZeroInt(), stored.VoterReward)
	k.False(stored.PendingExecution)
}
