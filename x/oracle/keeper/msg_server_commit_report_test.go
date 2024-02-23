package keeper_test

import (
	"encoding/hex"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/testutil/sample"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func (s *KeeperTestSuite) TestCommitValue() (reportertypes.OracleReporter, string) {
	// get the current query in cycle list
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	// value 100000000000000000000 in hex
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()

	stakedReporter := reportertypes.NewOracleReporter(
		addr.String(),
		math.NewInt(1_000_000),
		nil,
	)

	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(&stakedReporter, nil)

	var commitreq = types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
		Hash:      hash,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)

	_hexxy, err := hex.DecodeString(queryData)
	s.Nil(err)

	commitValue, err := s.oracleKeeper.Commits.Get(s.ctx, collections.Join(addr.Bytes(), keeper.HashQueryData(_hexxy)))
	s.NoError(err)
	s.Equal(true, s.oracleKeeper.VerifyCommit(s.ctx, addr.String(), value, salt, hash))
	s.Equal(commitValue.Report.Creator, addr.String())
	return stakedReporter, salt
}

func (s *KeeperTestSuite) TestCommitQueryNotInCycleList() {

	queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	// Commit report transaction
	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()

	stakedReporter := reportertypes.NewOracleReporter(
		addr.String(),
		math.NewInt(1_000_000),
		nil,
	)
	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(&stakedReporter, nil)

	var commitreq = types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
		Hash:      hash,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "query does not have tips and is not in cycle")
}

func (s *KeeperTestSuite) TestCommitQueryInCycleListPlusTippedQuery() {
	// commit query in cycle list
	queryData1, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	// Commit report transaction
	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()

	stakedReporter := reportertypes.NewOracleReporter(
		addr.String(),
		math.NewInt(1_000_000),
		nil,
	)
	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(&stakedReporter, nil)

	var commitreq = types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData1,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)

	// commit for query that was tipped
	queryData2 := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	tip := sdk.NewCoin("loya", math.NewInt(1000))
	_ = s.bankKeeper.On("SendCoinsFromAccountToModule", s.ctx, addr, types.ModuleName, sdk.NewCoins(tip)).Return(nil)
	// mock the 2% burn
	burnAmount := tip.Amount.MulRaw(2).QuoRaw(100)
	burned := sdk.NewCoin("loya", burnAmount)
	_ = s.bankKeeper.On("BurnCoins", s.ctx, types.ModuleName, sdk.NewCoins(burned)).Return(nil)

	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: queryData2,
		Amount:    tip,
	}
	_, err = s.msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	// commir for tipped query
	commitreq.Creator = addr.String()
	commitreq.QueryData = queryData2
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)

}

func (s *KeeperTestSuite) TestCommitWithBadQueryData() {

	// try to commit bad query data
	queryData := "stupidQueryData"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()

	stakedReporter := reportertypes.NewOracleReporter(
		addr.String(),
		math.NewInt(1_000_000),
		nil,
	)
	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(&stakedReporter, nil)

	var commitreq = types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
		Hash:      hash,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "invalid query data")
}

func (s *KeeperTestSuite) TestCommitWithReporterWithLowStake() {
	// try to commit from unbonded reporter
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)

	randomAddr := sample.AccAddressBytes()

	stakedReporter := reportertypes.NewOracleReporter(
		randomAddr.String(),
		math.NewInt(1), // below the min stake amount
		nil,
	)
	_ = s.reporterKeeper.On("Reporter", s.ctx, randomAddr).Return(&stakedReporter, nil)

	var commitreq = types.MsgCommitReport{
		Creator:   randomAddr.String(),
		QueryData: queryData,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "reporter has 1, required 1000000: not enough stake")
}

func (s *KeeperTestSuite) TestCommitWithJailedValidator() {
	// try to commit from jailed reporter
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)

	randomAddr := sample.AccAddressBytes()

	stakedReporter := reportertypes.NewOracleReporter(
		randomAddr.String(),
		math.NewInt(1_000_000),
		nil,
	)
	stakedReporter.Jailed = true
	_ = s.reporterKeeper.On("Reporter", s.ctx, randomAddr).Return(&stakedReporter, nil)

	s.Equal(true, stakedReporter.Jailed)

	var commitreq = types.MsgCommitReport{
		Creator:   randomAddr.String(),
		QueryData: queryData,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "reporter is jailed")

}

func (s *KeeperTestSuite) TestCommitWithMissingCreator() {
	// commit with no creator
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)
	s.Nil(err)

	var commitreq = types.MsgCommitReport{
		QueryData: queryData,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "invalid creator address")
}

func (s *KeeperTestSuite) TestCommitWithMissingQueryData() {
	// commit with no query data
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(value, salt)
	s.Nil(err)

	addr := sample.AccAddressBytes()

	var commitreq = types.MsgCommitReport{
		Creator: addr.String(),
		Hash:    hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "query data field cannot be empty")

}

func (s *KeeperTestSuite) TestCommitWithMissingHash() {
	// commit with no hash
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)

	addr := sample.AccAddressBytes()

	var commitreq = types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq) // no error
	s.ErrorContains(err, "hash field cannot be empty")
}

// // todo: check emitted events
