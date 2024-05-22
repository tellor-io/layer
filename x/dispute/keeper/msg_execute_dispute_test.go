package keeper_test

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k *KeeperTestSuite) TestExecuteDispute() {
	msgCreator := sample.AccAddressBytes()
	msg := types.MsgExecuteDispute{
		CallerAddress: msgCreator.String(),
		DisputeId:     1,
	}

	feePayer := sample.AccAddressBytes()

	dispute := k.dispute()
	dispute.BurnAmount = math.NewInt(500)
	dispute.PrevDisputeIds = []uint64{1}
	dispute.FeePayers = []types.PayerInfo{{PayerAddress: feePayer, Amount: math.NewInt(250)}}
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, 1, dispute))

	vote := types.Vote{
		Id: 1,
	}

	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, 1, vote))
	resp, err := k.msgServer.ExecuteDispute(k.ctx, &msg)
	k.Error(err, "can't execute, dispute not resolved")
	k.Nil(resp)

	dispute.DisputeStatus = types.Resolved
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, 1, dispute))
	resp, err = k.msgServer.ExecuteDispute(k.ctx, &msg)
	k.Error(err, "vote hasn't been tallied yet")
	k.Nil(resp)

	voter := sample.AccAddressBytes()

	modulAddr := authtypes.NewModuleAddress(types.ModuleName)
	k.bankKeeper.On("BurnCoins", k.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(250)))).Return(nil)
	k.accountKeeper.On("GetModuleAddress", types.ModuleName).Return(modulAddr, nil)
	k.bankKeeper.On("InputOutputCoins", k.ctx, banktypes.Input{Address: modulAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(250)))}, []banktypes.Output{{Address: voter.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(250)))}}).Return(nil, nil)
	k.bankKeeper.On("InputOutputCoins", k.ctx, banktypes.Input{Address: modulAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(9500)))}, []banktypes.Output{{Address: feePayer.String(), Coins: sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(9500)))}}).Return(nil, nil)
	k.reporterKeeper.On("AddAmountToStake", k.ctx, feePayer, math.NewInt(10000)).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, "bonded_tokens_pool", sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(10000)))).Return(nil)

	vote.VoteResult = types.VoteResult_SUPPORT
	k.NoError(k.disputeKeeper.Votes.Set(k.ctx, 1, vote))

	k.NoError(k.disputeKeeper.Voter.Set(k.ctx, collections.Join(uint64(1), voter.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.OneInt()}))
	resp, err = k.msgServer.ExecuteDispute(k.ctx, &msg)
	k.NoError(err)
	k.NotNil(resp)
}
