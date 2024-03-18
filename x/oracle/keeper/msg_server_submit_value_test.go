package keeper_test

import (
	"encoding/hex"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"

	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func (s *KeeperTestSuite) TestSubmitValue() (reportertypes.OracleReporter, string) {
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	// Commit value transaction first
	stakedReporter, salt, queryData := s.TestCommitValue()
	// forward block by 1 and reveal value
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	// Submit value transaction with value revealed, this checks if the value is correctly hashed
	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.MustAccAddressFromBech32(stakedReporter.GetReporter())).Return(&stakedReporter, nil)
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	var submitreq = types.MsgSubmitValue{
		Creator:   stakedReporter.GetReporter(),
		QueryData: queryData,
		Value:     value,
		Salt:      salt,
	}
	res, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.Nil(err)
	s.Equal(&types.MsgSubmitValueResponse{}, res)

	queryId, err := utils.QueryIDFromDataString(queryData)
	s.NoError(err)
	queryIdStr := hex.EncodeToString(queryId)

	report, err := s.queryClient.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: queryIdStr})
	s.Nil(err)

	microReport := types.MicroReport{
		Reporter:        stakedReporter.GetReporter(),
		Power:           stakedReporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64(),
		QueryType:       "SpotPrice",
		QueryId:         queryIdStr,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
	}
	expectedReport := types.QueryGetReportsbyQidResponse{
		Reports: types.Reports{
			MicroReports: []*types.MicroReport{&microReport},
		},
	}
	s.Equal(&expectedReport, report)

	return stakedReporter, queryIdStr
}

// Note: this test fails because logic allows for submit value with no commit
// func (s *KeeperTestSuite) TestSubmitFromWrongAddr() {

// 	// submit from different address than commit
// 	randomAddr := sample.AccAddressBytes()

// 	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// 	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

// 	stakedReporter, salt := s.TestCommitValue()
// 	stakedReporter.Reporter = randomAddr.String()

// 	var submitreq = types.MsgSubmitValue{
// 		Creator:   randomAddr.String(),
// 		QueryData: queryData,
// 		Value:     value,
// 		Salt:      salt,
// 	}

// 	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

// 	_ = s.reporterKeeper.On("Reporter", s.ctx, randomAddr).Return(&stakedReporter, nil)
// 	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)

// 	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
// 	s.Error(err)
// }

func (s *KeeperTestSuite) TestSubmitWithBadQueryData() {

	// submit value with bad query data
	badQueryData := "stupidQueryData"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	stakedReporter, salt, _ := s.TestCommitValue()

	var submitreq = types.MsgSubmitValue{
		Creator:   stakedReporter.GetReporter(),
		QueryData: badQueryData,
		Value:     value,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.MustAccAddressFromBech32(stakedReporter.GetReporter())).Return(&stakedReporter, nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "failed to decode query data string")
}

func (s *KeeperTestSuite) TestSubmitWithBadValue() {

	// submit wrong value but correct salt

	badValue := "00000F4240"

	stakedReporter, salt, queryData := s.TestCommitValue()

	var submitreq = types.MsgSubmitValue{
		Creator:   stakedReporter.GetReporter(),
		QueryData: queryData,
		Value:     badValue,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.MustAccAddressFromBech32(stakedReporter.GetReporter())).Return(&stakedReporter, nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

func (s *KeeperTestSuite) TestSubmitWithWrongSalt() {

	// submit correct value but wrong salt
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	stakedReporter, _, queryData := s.TestCommitValue()

	badSalt, err := oracleutils.Salt(32)
	s.Nil(err)

	var submitreq = types.MsgSubmitValue{
		Creator:   stakedReporter.GetReporter(),
		QueryData: queryData,
		Value:     value,
		Salt:      badSalt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.MustAccAddressFromBech32(stakedReporter.GetReporter())).Return(&stakedReporter, nil)

	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

func (s *KeeperTestSuite) TestSubmitAtWrongBlock() {

	// try to submit value in same block as commit
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	stakedReporter, salt, queryData := s.TestCommitValue()

	var submitreq = types.MsgSubmitValue{
		Creator:   stakedReporter.GetReporter(),
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
	_ = s.reporterKeeper.On("Reporter", s.ctx, sdk.MustAccAddressFromBech32(stakedReporter.GetReporter())).Return(&stakedReporter, nil) // submitreq.Salt = salt

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "missed commit reveal window")

}

// Note: no longer relevant since you can reveal without commit

// func (s *KeeperTestSuite) TestSubmitWithNoCommit() {

// 	// try to submit value without commit
// 	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
// 	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
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
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	_, salt, queryData := s.TestCommitValue()

	var submitreq = types.MsgSubmitValue{
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
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	stakedReporter, salt, _ := s.TestCommitValue()

	var submitreq = types.MsgSubmitValue{
		Creator: stakedReporter.GetReporter(),
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

	var submitreq = types.MsgSubmitValue{
		Creator:   stakedReporter.GetReporter(),
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
// 	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
// 	stakedReporter, _, queryData := s.TestCommitValue()
// 	var submitreq = types.MsgSubmitValue{
// 		Creator:   stakedReporter.GetReporter(),
// 		QueryData: queryData,
// 		Value:     value,
// 	}
// 	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
// 	s.ErrorContains(err, "salt cannot be empty")

// }
