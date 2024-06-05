package keeper_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestSubmitValue() (reportertypes.OracleReporter, []byte) {
	require := s.Require()
	// Commit
	stakedReporter, salt, queryData := s.TestCommitValue()
	// forward block by 1 and reveal value
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	// Submit value transaction with value revealed, this checks if the value is correctly hashed
	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.AccAddress(stakedReporter.GetReporter())).Return(&stakedReporter, nil)
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	submitreq := types.MsgSubmitValue{
		Creator:   sdk.AccAddress(stakedReporter.GetReporter()).String(),
		QueryData: queryData,
		Value:     value,
		Salt:      salt,
	}
	res, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	require.NoError(err)
	require.Equal(&types.MsgSubmitValueResponse{}, res)

	queryId := utils.QueryIDFromData(queryData)
	report, err := s.queryClient.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: hex.EncodeToString(queryId)})
	s.Nil(err)

	microReport := types.MicroReport{
		Reporter:        sdk.AccAddress(stakedReporter.GetReporter()).String(),
		Power:           stakedReporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryType:       "SpotPrice",
		QueryId:         queryId,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     s.ctx.BlockHeight(),
	}
	expectedReport := types.QueryGetReportsbyQidResponse{
		Reports: types.Reports{
			MicroReports: []*types.MicroReport{&microReport},
		},
	}
	require.Equal(&expectedReport, report)

	return stakedReporter, queryId
}

func (s *KeeperTestSuite) TestSubmitWithBadQueryData() {
	// submit value with bad query data
	badQueryData := []byte("stupidQueryData")

	stakedReporter, salt, _ := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   sdk.AccAddress(stakedReporter.GetReporter()).String(),
		QueryData: badQueryData,
		Value:     value,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.AccAddress(stakedReporter.GetReporter())).Return(&stakedReporter, nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "invalid query data")
}

func (s *KeeperTestSuite) TestSubmitWithBadValue() {
	// submit wrong value but correct salt

	badValue := "00000F4240"

	stakedReporter, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   sdk.AccAddress(stakedReporter.GetReporter()).String(),
		QueryData: queryData,
		Value:     badValue,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.AccAddress(stakedReporter.GetReporter())).Return(&stakedReporter, nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

func (s *KeeperTestSuite) TestSubmitWithWrongSalt() {
	// submit correct value but wrong salt

	stakedReporter, _, queryData := s.TestCommitValue()

	badSalt, err := oracleutils.Salt(32)
	s.Nil(err)

	submitreq := types.MsgSubmitValue{
		Creator:   sdk.AccAddress(stakedReporter.GetReporter()).String(),
		QueryData: queryData,
		Value:     value,
		Salt:      badSalt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.AccAddress(stakedReporter.GetReporter())).Return(&stakedReporter, nil)

	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

func (s *KeeperTestSuite) TestSubmitAtWrongBlock() {
	// try to submit value in same block as commit

	stakedReporter, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   sdk.AccAddress(stakedReporter.GetReporter()).String(),
		QueryData: queryData,
		Value:     value,
		Salt:      salt,
	}
	// Note: No longer relevant since you can reveal early
	// _, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	// s.ErrorContains(err, "commit reveal window is too early")

	// try to submit value 2 blocks after commit
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour))
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.AccAddress(stakedReporter.GetReporter())).Return(&stakedReporter, nil) // submitreq.Salt = salt

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "missed commit reveal window")
}

// Note: no longer relevant since you can reveal without commit

// func (s *KeeperTestSuite) TestSubmitWithNoCommit() {

// 	// try to submit value without commit
// 	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// 	salt, err := oracleutils.Salt(32)
// 	s.Nil(err)

// 	addr := sample.AccAddressBytes()

// 	var submitreq = types.MsgSubmitValue{
// 		Creator:   addr.String(),
// 		QueryData: queryData,
// 		Value:     value,
// 		Salt:      salt,
// 	}
// 	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour))

// 	stakedReporter := reportertypes.NewOracleReporter(
// 		addr.String(),
// 		math.NewInt(1_000_000),
// 		nil,
// 	)
// 	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(&stakedReporter, nil)

// 	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
// 	s.ErrorContains(err, "no commits to reveal found")
// }

func (s *KeeperTestSuite) TestSubmitWithNoCreator() {
	// submit value with no creator

	_, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		QueryData: queryData,
		Value:     value,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "invalid creator address")
}

func (s *KeeperTestSuite) TestSubmitWithNoQueryData() {
	// submit value with no query data

	stakedReporter, salt, _ := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator: sdk.AccAddress(stakedReporter.GetReporter()).String(),
		Value:   value,
		Salt:    salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "query data cannot be empty")
}

func (s *KeeperTestSuite) TestSubmitWithNoValue() {
	// submit value with no value
	stakedReporter, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   sdk.AccAddress(stakedReporter.GetReporter()).String(),
		QueryData: queryData,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "value cannot be empty")
}

// Note: this test fails because logic allows for submit value with no salt

// func (s *KeeperTestSuite) TestSubmitWithbadSalt() {
// 	// submit value with no salt
// 	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// 	stakedReporter, _, queryData := s.TestCommitValue()
// 	var submitreq = types.MsgSubmitValue{
// 		Creator:   stakedReporter.GetReporter(),
// 		QueryData: queryData,
// 		Value:     value,
// 	}
// 	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
// 	s.ErrorContains(err, "salt cannot be empty")

// }
