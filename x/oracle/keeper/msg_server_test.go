package keeper_test

import (
	"context"
	"encoding/hex"
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/mocks"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, *keeper.Keeper, *mocks.StakingKeeper, *mocks.AccountKeeper, context.Context) {
	k, sk, ak, ctx := keepertest.OracleKeeper(t)
	return keeper.NewMsgServerImpl(*k), k, sk, ak, sdk.WrapSDKContext(ctx)
}

func TestMsgServer(t *testing.T) {
	ms, _, _, _, goctx := setupMsgServer(t)

	require.NotNil(t, ms)
	require.NotNil(t, goctx)
}
func KeyTestPubAddr() (cryptotypes.PrivKey, cryptotypes.PubKey, sdk.AccAddress) {
	key := secp256k1.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
func TestSubmitValue(t *testing.T) {
	ms, k, sk, ak, goctx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goctx)
	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	var commitreq types.MsgCommitReport
	// var commitres *types.MsgCommitReportResponse
	var submitreq types.MsgSubmitValue
	var submitres types.MsgSubmitValueResponse
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
	_, err = ms.CommitReport(goctx, &commitreq)
	require.Nil(t, err)
	// forward block by 1 and reveal value
	height := ctx.BlockHeight() + 1
	ctx = ctx.WithBlockHeight(height)
	// Submit value transaction with value revealed, this checks if the value is correctly signed
	submitreq.Creator = addr.String()
	submitreq.QueryData = queryData
	submitreq.Value = value

	delegation := stakingtypes.Delegation{
		DelegatorAddress: addr.String(),
		ValidatorAddress: "",
		Shares:           sdk.NewDec(100),
	}
	sk.On("GetAllDelegatorDelegations", mock.Anything, mock.Anything).Return([]stakingtypes.Delegation{delegation}, nil)
	account := authtypes.NewBaseAccount(addr, pub, 0, 0)
	ak.On("GetAccount", mock.Anything, mock.Anything).Return(account, nil)
	res, err := ms.SubmitValue(ctx, &submitreq)
	require.Equal(t, &submitres, res)
	require.Nil(t, err)
	report, err := k.GetReportsbyQid(ctx, &types.QueryGetReportsbyQidRequest{QId: "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"})
	require.Nil(t, err)
	microReport := types.MicroReport{
		Reporter:  addr.String(),
		Qid:       "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     value,
		Timestamp: uint64(ctx.BlockTime().Unix()),
	}
	expectedReport := types.QueryGetReportsbyQidResponse{
		Reports: types.Reports{
			MicroReports: []*types.MicroReport{&microReport},
		},
	}
	require.Equal(t, &expectedReport, report)
}
