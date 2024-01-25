package keeper_test

import (
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	utils "github.com/tellor-io/layer/x/oracle/utils"
)

func (s *KeeperTestSuite) TestCommitValue() {
	require := s.Require()
	queryData := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	salt, err := utils.Salt(32)
	fmt.Println("salt", salt)
	require.Nil(err)
	// signature, err := PrivKey.Sign(valueDecoded)
	saltedValue := utils.CalculateCommitment(string(valueDecoded), salt)
	fmt.Println("saltedValue", saltedValue)
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
}

func (s *KeeperTestSuite) TestCommitQueryNotInCycleList() {
	require := s.Require()
	queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	// signature, err := PrivKey.Sign(valueDecoded)
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
	queryData1 := s.oracleKeeper.GetCurrentQueryInCycleList(s.ctx)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	valueDecoded, err := hex.DecodeString(value)
	s.Nil(err)
	// signature, err := PrivKey.Sign(valueDecoded)
	salt, err := utils.Salt(32)
	s.Nil(err)
	saltedValue := utils.CalculateCommitment(string(valueDecoded), salt)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData1
	commitreq.SaltedValue = saltedValue
	_, err = s.msgServer.CommitReport(s.ctx, &commitreq)
	s.Nil(err)

}
