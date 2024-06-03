package keeper_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestCommitValue() (sdk.AccAddress, string, []byte) {
	// get the current query in cycle list
	s.ctx = s.ctx.WithBlockTime(time.Now())
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()

	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	commitreq := types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
		Hash:      hash,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.Nil(err)

	qId := utils.QueryIDFromData(queryData)
	query, err := s.oracleKeeper.Query.Get(s.ctx, qId)
	s.Nil(err)
	s.NotNil(query)
	commitValue, err := s.oracleKeeper.Commits.Get(s.ctx, collections.Join(addr.Bytes(), query.Id))
	s.Nil(err)
	s.Equal(true, s.oracleKeeper.VerifyCommit(s.ctx, addr.String(), value, salt, hash))
	s.Equal(commitValue.Reporter, addr.String())
	return addr, salt, queryData
}

func (s *KeeperTestSuite) TestCommitQueryNotInCycleList() {
	queryData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")

	// Commit report transaction
	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()

	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	commitreq := types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
		Hash:      hash,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "query not part of cyclelist")
}

func (s *KeeperTestSuite) TestCommitQueryInCycleListPlusTippedQuery() {
	s.ctx = s.ctx.WithBlockTime(time.Now())
	// commit query in cycle list
	queryData1, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)

	// Commit report transaction
	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)

	addr := sample.AccAddressBytes()
	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	_ = s.reporterKeeper.On("Reporter", s.ctx, addr).Return(math.NewInt(1_000_000), nil)

	commitreq := types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData1,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)

	// commit for query that was tipped
	queryData2, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
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

func (s *KeeperTestSuite) TestCommitWithReporterWithLowStake() {
	// try to commit from unbonded reporter
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)

	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)

	randomAddr := sample.AccAddressBytes()

	_ = s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(registrytypes.GenesisDataSpec(), nil)
	_ = s.reporterKeeper.On("Reporter", s.ctx, randomAddr).Return(math.OneInt(), nil)

	commitreq := types.MsgCommitReport{
		Creator:   randomAddr.String(),
		QueryData: queryData,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "reporter has 1, required amount is 1000000: not enough stake")
}

func (s *KeeperTestSuite) TestCommitWithJailedValidator() {
	// try to commit from jailed reporter
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)

	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)

	randomAddr := sample.AccAddressBytes()

	_ = s.reporterKeeper.On("Reporter", s.ctx, randomAddr).Return(math.Int{}, reportertypes.ErrReporterJailed)

	commitreq := types.MsgCommitReport{
		Creator:   randomAddr.String(),
		QueryData: queryData,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "reporter jailed")
}

func (s *KeeperTestSuite) TestCommitWithMissingCreator() {
	// commit with no creator
	queryData, err := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)

	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)
	s.Nil(err)

	commitreq := types.MsgCommitReport{
		QueryData: queryData,
		Hash:      hash,
	}

	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.ErrorContains(err, "invalid creator address")
}

func (s *KeeperTestSuite) TestCommitWithMissingQueryData() {
	// commit with no query data

	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)
	s.Nil(err)

	addr := sample.AccAddressBytes()

	commitreq := types.MsgCommitReport{
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

	commitreq := types.MsgCommitReport{
		Creator:   addr.String(),
		QueryData: queryData,
	}
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq) // no error
	s.ErrorContains(err, "hash field cannot be empty")
}
