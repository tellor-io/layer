package app_test

import (
	"fmt"
	"os"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type VoteExtensionTestSuite struct {
	suite.Suite
	ctx     sdk.Context
	handler *app.VoteExtHandler
}

func (s *VoteExtensionTestSuite) SetupTest() {
	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", os.TempDir())
	viper.Set("key-name", "my-key-name")

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	s.handler = app.NewVoteExtHandler(
		log.NewNopLogger(),
		cdc,
		mocks.NewOracleKeeper(s.T()),
		mocks.NewBridgeKeeper(s.T()),
	)
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

func (s *VoteExtensionTestSuite) TestGetKeyring() {
	require := s.Require()
	h := s.handler

	kr, err := h.GetKeyring()
	require.NoError(err)
	require.NotNil(kr)

}

func (s *VoteExtensionTestSuite) TestGetOperatorAddress() {
	require := s.Require()
	h := s.handler
	kr, err := h.GetKeyring()
	require.NoError(err)
	testutils.ClearKeyring(s.T(), kr)

	key, mnemonic, err := kr.NewMnemonic("key-1", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	require.NoError(err)
	require.NotNil(key)
	hdPath := "m/44'/118'/0'/0/0" // BIP-44 path for Cosmos SDK
	record, err := kr.NewAccount("my-key-name", mnemonic, "", hdPath, hd.Secp256k1)
	fmt.Println("record: ", record)
	fmt.Println("key: ", key)
	fmt.Println("mnemonic: ", mnemonic)
	fmt.Println("kr: ", kr)
	require.NoError(err)
	require.NotNil(record)
	addr, err := h.GetOperatorAddress()
	require.NoError(err)
	require.NotNil(addr)
}

func (s *VoteExtensionTestSuite) TestExtendVoteHandler() {
	require := s.Require()
	h := s.handler

	res, err := h.ExtendVoteHandler(s.ctx, &abci.RequestExtendVote{})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res, &abci.ResponseExtendVote{})
}
