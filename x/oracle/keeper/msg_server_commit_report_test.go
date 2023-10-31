package keeper_test

import (
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestCommitReport() {
	require := s.Require()
	queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	var commitRes types.MsgCommitReportResponse
	// generate keys
	key, pub, addr := KeyTestPubAddr()
	// Commit report transaction
	commitreq.Creator = addr.String()
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	commitreq.QueryData = queryData
	signature, err := key.Sign(valueDecoded)
	require.Nil(err)
	commitreq.Signature = hex.EncodeToString(signature)
	res, err := s.msgServer.CommitReport(sdk.WrapSDKContext(s.ctx), &commitreq)
	height := s.ctx.BlockHeight() + 1
	ctx := s.ctx.WithBlockHeight(height)
	require.Nil(err)
	require.Equal(&commitRes, res)
	_hexxy, _ := hex.DecodeString(queryData)
	commitValue, err := s.oracleKeeper.GetSignature(ctx, addr.String(), keeper.HashQueryData(_hexxy))
	_addy, _ := sdk.AccAddressFromBech32(commitreq.Creator)
	account := authtypes.NewBaseAccount(addr, pub, 0, 0)
	s.accountKeeper.On("GetAccount", mock.Anything, mock.Anything).Return(account, nil)
	require.Equal(true, s.oracleKeeper.VerifySignature(ctx, _addy, value, commitValue.Report.Signature))
	require.Equal(commitValue.Report.Creator, addr.String())
}
