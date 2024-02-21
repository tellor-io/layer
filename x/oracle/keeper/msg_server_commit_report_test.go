package keeper_test

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	utils "github.com/tellor-io/layer/x/oracle/utils"
)

func (s *KeeperTestSuite) TestCommitValue() string {
	require := s.Require()

	queryData := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(value, salt)
	require.Nil(err)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)
	_hexxy, _ := hex.DecodeString(queryData)
	commitValue, err := s.oracleKeeper.Commits.Get(s.ctx, collections.Join(Addr.Bytes(), keeper.HashQueryData(_hexxy)))
	s.NoError(err)
	require.Equal(true, s.oracleKeeper.VerifyCommit(s.ctx, Addr.String(), value, salt, hash))
	require.Equal(commitValue.Report.Creator, Addr.String())
	return salt
}

func (s *KeeperTestSuite) TestCommitQueryNotInCycleList() {
	require := s.Require()

	queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	require.ErrorContains(err, "query data does not have tips/not in cycle")
}

func (s *KeeperTestSuite) TestCommitQueryInCycleListPlusTippedQuery() {
	// commit query in cycle list
	queryData1 := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	valueDecoded, err := hex.DecodeString(value)
	s.Nil(err)
	salt, err := utils.Salt(32)
	s.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData1
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)

	// commit for query that was tipped
	queryData2 := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	tip := sdk.NewCoin("loya", math.NewInt(1000))
	msg := types.MsgTip{
		Tipper:    Addr.String(),
		QueryData: queryData2,
		Amount:    tip,
	}
	_, err = s.msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData2
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)

}

func (s *KeeperTestSuite) TestCommitWithBadQueryData() {
	require := s.Require()

	// try to commit bad query data
	queryData := "stupidQueryData"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	require.ErrorContains(err, "invalid query data")
}

func (s *KeeperTestSuite) TestCommitWithUnbondedValidator() {
	require := s.Require()

	// try to commit from unbonded validator
	queryData := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	randomPrivKey := secp256k1.GenPrivKey()
	randomPubKey := randomPrivKey.PubKey()
	randomAddr := sdk.AccAddress(randomPubKey.Address())
	badValidator, _ := stakingtypes.NewValidator(randomAddr.String(), randomPubKey, stakingtypes.Description{Moniker: "unbonded badguy"})
	badValidator.Jailed = false
	badValidator.Status = stakingtypes.Unbonded
	badValidator.Tokens = math.NewInt(1000000000000000000)
	require.Equal(false, badValidator.IsBonded())
	require.Equal(false, badValidator.IsJailed())
	s.stakingKeeper.ExpectedCalls = []*mock.Call{}
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(badValidator, nil)
	s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(randomAddr))
	commitreq.Creator = randomAddr.String()
	commitreq.QueryData = queryData
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	require.ErrorContains(err, "validator is not bonded")
}

func (s *KeeperTestSuite) TestCommitWithJailedValidator() {
	require := s.Require()

	// try to commit from jailed validator
	queryData := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	randomPrivKey := secp256k1.GenPrivKey()
	randomPubKey := randomPrivKey.PubKey()
	randomAddr := sdk.AccAddress(randomPubKey.Address())
	badValidator, _ := stakingtypes.NewValidator(randomAddr.String(), randomPubKey, stakingtypes.Description{Moniker: "jailed badguy"})
	badValidator.Jailed = true
	badValidator.Status = stakingtypes.Bonded
	badValidator.Tokens = math.NewInt(1000000000000000000)
	s.stakingKeeper.ExpectedCalls = []*mock.Call{}
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(badValidator, nil)
	s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(badValidator.OperatorAddress))
	require.Equal(true, badValidator.IsJailed())
	require.Equal(true, badValidator.IsBonded())
	commitreq.Creator = randomAddr.String()
	commitreq.QueryData = queryData
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	require.ErrorContains(err, "validator is jailed")

}

func (s *KeeperTestSuite) TestCommitWithMissingCreator() {
	require := s.Require()

	// commit with no creator
	queryData := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	require.Nil(err)
	commitreq.QueryData = queryData
	commitreq.Hash = hash
	require.Panics(func() { s.msgServer.CommitReport(s.ctx, &commitreq) }, "empty address string is not allowed")
}

// Should ppl be allowed to commit with no query data and/or no hash ?
func (s *KeeperTestSuite) TestCommitWithMissingQueryData() {
	require := s.Require()

	// commit with no query data
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	hash := utils.CalculateCommitment(string(valueDecoded), salt)
	require.Nil(err)
	commitreq.Creator = Addr.String()
	commitreq.Hash = hash
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq) // no error
	fmt.Println(err)

}

func (s *KeeperTestSuite) TestCommitWithMissingHash() {

	// commit with no hash
	queryData := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	var commitreq types.MsgCommitReport
	commitreq.QueryData = queryData
	commitreq.Creator = Addr.String()
	_, err := s.msgServer.CommitReport(s.ctx, &commitreq) // no error
	fmt.Println(err)
}

// todo: check emitted events
