package keeper_test

import (
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestSubmitValue() {
	require := s.Require()
	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// var commitres *types.MsgCommitReportResponse
	var submitreq types.MsgSubmitValue
	var submitres types.MsgSubmitValueResponse
	// Commit report transaction
	commitreq.Creator = Addr.String()
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	commitreq.QueryData = queryData
	signature, err := PrivKey.Sign(valueDecoded)
	require.Nil(err)
	commitreq.Signature = hex.EncodeToString(signature)
	_, err = s.msgServer.CommitReport(sdk.WrapSDKContext(s.ctx), &commitreq)
	require.Nil(err)
	// forward block by 1 and reveal value
	height := s.ctx.BlockHeight() + 1
	ctx := s.ctx.WithBlockHeight(height)
	// Submit value transaction with value revealed, this checks if the value is correctly signed
	submitreq.Creator = Addr.String()
	submitreq.QueryData = queryData
	submitreq.Value = value
	delegation := stakingtypes.Delegation{
		DelegatorAddress: Addr.String(),
		ValidatorAddress: "",
		Shares:           sdk.NewDec(100),
	}
	s.stakingKeeper.On("GetAllDelegatorDelegations", mock.Anything, mock.Anything).Return([]stakingtypes.Delegation{delegation}, nil)
	account := authtypes.NewBaseAccount(Addr, PubKey, 0, 0)
	s.accountKeeper.On("GetAccount", mock.Anything, mock.Anything).Return(account, nil)
	res, err := s.msgServer.SubmitValue(ctx, &submitreq)
	require.Equal(&submitres, res)
	require.Nil(err)
	report, err := s.oracleKeeper.GetReportsbyQid(ctx, &types.QueryGetReportsbyQidRequest{QId: "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"})
	require.Nil(err)
	microReport := types.MicroReport{
		Reporter:  Addr.String(),
		Qid:       "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     value,
		Timestamp: uint64(ctx.BlockTime().Unix()),
	}
	expectedReport := types.QueryGetReportsbyQidResponse{
		Reports: types.Reports{
			MicroReports: []*types.MicroReport{&microReport},
		},
	}
	require.Equal(&expectedReport, report)
}
