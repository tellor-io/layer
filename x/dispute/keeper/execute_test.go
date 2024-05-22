package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k *KeeperTestSuite) TestExecuteVote() {
	dispute := k.dispute()

	// slash amount = 10000
	dispute.FeePayers = []types.PayerInfo{
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(800), FromBond: true},
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(200), FromBond: false},
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

	modulAddr := authtypes.NewModuleAddress(types.ModuleName)
	k.ctx = k.ctx.WithBlockTime(k.ctx.BlockTime().Add(1))

	// mocks
	k.bankKeeper.On("BurnCoins", k.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("loya", math.ZeroInt()))).Return(nil)
	k.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(modulAddr, nil)
	k.reporterKeeper.On("FeeRefund", k.ctx, sdk.AccAddress(dispute.FeePayers[0].PayerAddress), dispute.HashId, math.NewInt(8000)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(8000)))).Return(nil)
	k.bankKeeper.On(
		"InputOutputCoins", k.ctx,
		banktypes.Input{Address: modulAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(2000)))},
		[]banktypes.Output{
			{Address: sdk.AccAddress(dispute.FeePayers[1].PayerAddress).String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(2000)))},
		}).Return(nil, nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(dispute.FeePayers[0].PayerAddress), math.NewInt(8000)).Return(nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(dispute.FeePayers[1].PayerAddress), math.NewInt(2000)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(10000)))).Return(nil)

	k.NoError(k.disputeKeeper.ExecuteVote(k.ctx, dispute.DisputeId))
}

func (k *KeeperTestSuite) TestRefundDisputeFee() {
	disputeFeeMinusBurn := math.NewInt(950)
	feePayers := []types.PayerInfo{
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(800), FromBond: true},
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(200), FromBond: false},
	}
	modulAddr := authtypes.NewModuleAddress(types.ModuleName)
	k.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(modulAddr, nil)
	k.reporterKeeper.On("FeeRefund", k.ctx, sdk.AccAddress(feePayers[0].PayerAddress), []byte("hash"), math.NewInt(760)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(760)))).Return(nil)
	k.bankKeeper.On(
		"InputOutputCoins", k.ctx,
		banktypes.Input{Address: modulAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(190)))},
		[]banktypes.Output{
			{Address: sdk.AccAddress(feePayers[1].PayerAddress).String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(190)))},
		}).Return(nil, nil)

	k.NoError(k.disputeKeeper.RefundDisputeFee(k.ctx, feePayers, disputeFeeMinusBurn, []byte("hash")))
}

func (k *KeeperTestSuite) TestRewardReporterBondToFeePayers() {
	reporterBond := math.NewInt(1000)
	feePayers := []types.PayerInfo{
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(8), FromBond: true},
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(2), FromBond: true},
	}
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(feePayers[0].PayerAddress), math.NewInt(800)).Return(nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(feePayers[1].PayerAddress), math.NewInt(200)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", reporterBond))).Return(nil)
	k.NoError(k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feePayers, reporterBond))

	feePayers = []types.PayerInfo{
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(8), FromBond: true},
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(2), FromBond: true},
		{PayerAddress: sample.AccAddressBytes(), Amount: math.NewInt(3), FromBond: true},
	}
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(feePayers[0].PayerAddress), math.NewInt(615)).Return(nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(feePayers[1].PayerAddress), math.NewInt(153)).Return(nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, sdk.AccAddress(feePayers[2].PayerAddress), math.NewInt(232)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", reporterBond))).Return(nil)
	k.NoError(k.disputeKeeper.RewardReporterBondToFeePayers(k.ctx, feePayers, reporterBond))
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
