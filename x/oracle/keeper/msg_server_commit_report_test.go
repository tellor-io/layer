package keeper_test

import (
	"encoding/hex"
	"fmt"

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
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	saltedValue := utils.CalculateCommitment(string(valueDecoded), salt)
	require.Nil(err)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.SaltedValue = saltedValue
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.NoError(err)
	_hexxy, _ := hex.DecodeString(queryData)
	commitValue, err := s.oracleKeeper.GetSignature(s.ctx, Addr, keeper.HashQueryData(_hexxy))
	s.NoError(err)
	fmt.Println("commitValue:", commitValue)
	fmt.Println("verify commit: ", s.oracleKeeper.VerifyCommit(s.ctx, Addr.String(), value, salt, saltedValue))

	require.Equal(true, s.oracleKeeper.VerifyCommit(s.ctx, Addr.String(), value, salt, saltedValue))
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
	saltedValue := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.SaltedValue = saltedValue
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
	saltedValue := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData1
	commitreq.SaltedValue = saltedValue
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.Nil(err)

	// commit query that was tipped

}

func (s *KeeperTestSuite) TestBadCommits() {
	// try to commit bad query data
	require := s.Require()
	queryData := "stupidQueryData"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	require.Nil(err)
	saltedValue := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.SaltedValue = saltedValue
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	require.ErrorContains(err, "invalid query data")

	// try to commit from unbonded validator
	randomPrivKey := secp256k1.GenPrivKey()
	randomPubKey := randomPrivKey.PubKey()
	randomAddr := sdk.AccAddress(randomPubKey.Address())
	badValidator, _ := stakingtypes.NewValidator(randomAddr.String(), randomPubKey, stakingtypes.Description{Moniker: "unbonded badguy"})
	badValidator.Jailed = false
	badValidator.Status = stakingtypes.Unbonded
	badValidator.Tokens = math.NewInt(1000000000000000000)
	require.Equal(false, badValidator.IsBonded())
	require.Equal(false, badValidator.IsJailed())
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(badValidator, nil)
	s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(randomAddr))
	queryData = s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	commitreq.Creator = badValidator.OperatorAddress
	commitreq.QueryData = queryData
	commitreq.SaltedValue = saltedValue
	fmt.Println("&commitreq:", &commitreq)
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	require.ErrorContains(err, "validator is not bonded")

	// try to commit from jailed validator
	randomPrivKey2 := secp256k1.GenPrivKey()
	randomPubKey2 := randomPrivKey2.PubKey()
	randomAddr2 := sdk.AccAddress(randomPubKey2.Address())
	badValidator, _ = stakingtypes.NewValidator(randomAddr2.String(), randomPubKey2, stakingtypes.Description{Moniker: "jailed badguy"})
	badValidator.Jailed = true
	badValidator.Status = stakingtypes.Bonded
	badValidator.Tokens = math.NewInt(10000000000000000)
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(badValidator, nil)
	s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(badValidator.OperatorAddress))
	require.Equal(true, badValidator.IsJailed())
	require.Equal(true, badValidator.IsBonded())
	commitreq.Creator = badValidator.OperatorAddress
	commitreq.QueryData = queryData
	commitreq.SaltedValue = saltedValue
	fmt.Println("&commitreq:", &commitreq)
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	fmt.Println("err:", err)
	require.ErrorContains(err, "validator is jailed") // fails bc err is "validator is not bonded" ?
}
