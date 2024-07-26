package app_test

import (
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
	require := s.Require()

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	s.ctx = testutils.CreateTestContext(s.T())
	s.handler = app.NewVoteExtHandler(
		log.NewNopLogger(),
		cdc,
		mocks.NewOracleKeeper(s.T()),
		mocks.NewBridgeKeeper(s.T()),
	)

	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", "~/.layer/my-key-name-5")
	viper.Set("key-name", "my-key-name-5")

	kr, err := s.handler.InitKeyring()
	require.NoError(err)
	require.NotNil(kr)

	// cmd := exec.Command("layerd", "keys", "add", "my-key-name-5")
	// output, err := cmd.CombinedOutput()
	// require.NoError(err)
	// require.NotNil(output)
	// fmt.Println(string(output))

	key, mnemonic, err := kr.NewMnemonic("my-key-name-4", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	require.NoError(err)
	require.NotNil(key)
	require.NotNil(mnemonic)
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

// create a keyring in the test, sotre in temp directort, modify voteexthandler

// read youre own computer

// create keyring the exact same way

// maybe even pass in a keyring instead of reading it

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
	// kr, err := h.GetKeyring()
	// require.NoError(err)

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
