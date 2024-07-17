package keeper_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestSubmitValue() (sdk.AccAddress, []byte) {
	require := s.Require()
	// Commit
	addr, salt, queryData := s.TestCommitValue()
	// forward block by 1 and reveal value
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	// Submit value transaction with value revealed, this checks if the value is correctly hashed
	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
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
		Reporter:        addr.String(),
		Power:           1,
		QueryType:       "SpotPrice",
		QueryId:         queryId,
		AggregateMethod: "weighted-median",
		Value:           value,
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     s.ctx.BlockHeight(),
	}
	expectedReport := types.QueryMicroReportsResponse{
		MicroReports: []types.MicroReport{microReport},
	}
	require.Equal(expectedReport.MicroReports, report.MicroReports)

	return addr, queryId
}

func (s *KeeperTestSuite) TestSubmitWithBadQueryData() {
	// submit value with bad query data
	badQueryData := []byte("stupidQueryData")

	addr, salt, _ := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: badQueryData,
		Value:     value,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "invalid query data")
}

func (s *KeeperTestSuite) TestSubmitWithBadValue() {
	// submit wrong value but correct salt

	badValue := "00000F4240"

	addr, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: queryData,
		Value:     badValue,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

func (s *KeeperTestSuite) TestSubmitWithWrongSalt() {
	// submit correct value but wrong salt
	addr, _, queryData := s.TestCommitValue()

	badSalt, err := oracleutils.Salt(32)
	s.Nil(err)

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: queryData,
		Value:     value,
		Salt:      badSalt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	_, err = s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "submitted value doesn't match commitment, are you a cheater?")
}

func (s *KeeperTestSuite) TestSubmitAtWrongBlock() {
	// try to submit value in same block as commit

	addr, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
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
	_ = s.reporterKeeper.On("ReporterStake", s.ctx, addr).Return(math.NewInt(1_000_000), nil) // submitreq.Salt = salt

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "missed commit reveal window")
}

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

	addr, salt, _ := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator: addr.String(),
		Value:   value,
		Salt:    salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "query data cannot be empty")
}

func (s *KeeperTestSuite) TestSubmitWithNoValue() {
	// submit value with no value
	addr, salt, queryData := s.TestCommitValue()

	submitreq := types.MsgSubmitValue{
		Creator:   addr.String(),
		QueryData: queryData,
		Salt:      salt,
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)

	_, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	s.ErrorContains(err, "value cannot be empty")
}
