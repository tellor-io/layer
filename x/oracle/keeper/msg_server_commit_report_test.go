package keeper_test

import (
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestCommitValue() {
	require := s.Require()
	queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq types.MsgCommitReport
	// Commit report transaction
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	signature, err := PrivKey.Sign(valueDecoded)
	require.Nil(err)
	commitreq.Creator = Addr.String()
	commitreq.QueryData = queryData
	commitreq.Signature = hex.EncodeToString(signature)
	addy, err := sdk.AccAddressFromBech32(commitreq.Creator)
	val, err := stakingtypes.NewValidator(sdk.ValAddress(addy), PubKey, stakingtypes.Description{Moniker: "test"})
	val.Jailed = false
	val.Status = stakingtypes.Bonded
	val.Tokens = sdk.NewInt(1000000000000000000)
	s.stakingKeeper.On("Validator", mock.Anything, mock.Anything).Return(val, true)
	_, err = s.msgServer.CommitReport(sdk.WrapSDKContext(s.ctx), &commitreq)
	require.Nil(err)
}
