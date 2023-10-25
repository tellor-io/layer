package keeper_test

import (
	"encoding/hex"
	"testing"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func TestCommitReport(t *testing.T) {
	ms, k, _,ak,goctx := setupMsgServer(t)
	queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	ctx := sdk.UnwrapSDKContext(goctx)
	var commitreq types.MsgCommitReport
	// var commitres *types.MsgCommitReportResponse
	//var commitReq types.MsgCommitReport
	var commitRes types.MsgCommitReportResponse
	// generate keys
	key, pub, addr := KeyTestPubAddr()
	// Commit report transaction
	commitreq.Creator = addr.String()
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(t, err)
	commitreq.QueryData = queryData
	signature, err := key.Sign(valueDecoded)
	require.Nil(t, err)
	commitreq.Signature = hex.EncodeToString(signature)
	res, err := ms.CommitReport(goctx, &commitreq)
	height := ctx.BlockHeight() + 1
	ctx = ctx.WithBlockHeight(height)
	require.Nil(t, err)
	require.Equal(t, &commitRes, res)
	_hexxy, _ := hex.DecodeString(queryData)
	commitValue, err := k.GetSignature(ctx, addr.String(), keeper.HashQueryData(_hexxy))
	_addy, _ := sdk.AccAddressFromBech32(commitreq.Creator)
	account := authtypes.NewBaseAccount(addr, pub, 0, 0)
	ak.On("GetAccount", mock.Anything, mock.Anything).Return(account, nil)
	require.Equal(t,true, k.VerifySignature(ctx, _addy, value, commitValue.Report.Signature))
	require.Equal(t,commitValue.Report.Creator, addr.String() )
}
