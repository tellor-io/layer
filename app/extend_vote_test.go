package app_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"
	"github.com/tellor-io/layer/testutil/sample"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type VoteExtensionTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	handler       *app.VoteExtHandler
	oracleKeeper  *mocks.OracleKeeper
	bridgeKeeper  *mocks.BridgeKeeper
	stakingKeeper *mocks.StakingKeeper
	kr            keyring.Keyring
	tempDir       string
	cdc           codec.Codec
}

func (s *VoteExtensionTestSuite) SetupTest() {
	require := s.Require()

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	s.cdc = cdc
	s.oracleKeeper = mocks.NewOracleKeeper(s.T())
	s.bridgeKeeper = mocks.NewBridgeKeeper(s.T())
	s.stakingKeeper = mocks.NewStakingKeeper(s.T())

	// create new vote ext handler
	s.ctx = testutils.CreateTestContext(s.T())
	s.handler = app.NewVoteExtHandler(
		log.NewNopLogger(),
		cdc,
		s.oracleKeeper,
		s.bridgeKeeper,
	)

	// create new keyring in
	s.tempDir = s.T().TempDir()
	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", s.tempDir)
	var err error
	s.kr, err = s.handler.InitKeyring()
	require.NoError(err)
	require.NotNil(s.kr)

	// backend := "test"
	// tempDir := s.T().TempDir()
	// keyring, err := keyring.New("TestExport", backend, tempDir, nil, cdc)
	// fmt.Println("keyring: ", keyring)
	// fmt.Println("temp dir: ", tempDir)
	// s.kr = keyring
	// require.NoError(err)

	// keys, err := s.kr.List()
	// require.NoError(err)
	// fmt.Println("Keys in keyring: ", keys)
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler() {
	require := s.Require()
	h := s.handler
	bk := s.bridgeKeeper

	// err unmarshalling, empty req.ValidatorAddress, err on GetEVMAddressByOperator, accept
	bk.On("GetEVMAddressByOperator", s.ctx, "").Return(nil, errors.New("error")).Once()
	res, err := h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{})
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_ACCEPT)

	// err unmarshalling, validator has evm address, val has EVM addr, reject
	req := &abci.RequestVerifyVoteExtension{
		ValidatorAddress: []byte("operatorIn"),
	}
	validatorAddress, err := sdk.Bech32ifyAddressBytes(sdk.GetConfig().GetBech32ValidatorAddrPrefix(), req.ValidatorAddress)
	require.NoError(err)
	bk.On("GetEVMAddressByOperator", s.ctx, validatorAddress).Return([]byte("operatorOut"), nil).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no err unmarshalling, not coll not found err from GetAttestationRequestsByHeight, reject
	s.ctx = s.ctx.WithBlockHeight(3)
	oracleAtt := app.OracleAttestation{
		Attestation: []byte("attestation"),
		Snapshot:    []byte("snapshot"),
	}
	bridgeVoteExt := &app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			oracleAtt,
		},
		InitialSignature: app.InitialSignature{
			SignatureA: []byte("signature"),
			SignatureB: []byte("signature"),
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: []byte("signature"),
			Timestamp: uint64(s.ctx.BlockTime().Unix()),
		},
	}
	bridgeVoteExtBz, err := json.Marshal(bridgeVoteExt)
	require.NoError(err)
	req = &abci.RequestVerifyVoteExtension{
		VoteExtension: bridgeVoteExtBz,
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(nil, errors.New("error")).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no err unmarshalling, coll not found err from GetAttestationRequestsByHeight, oracle att length > 0, reject
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(nil, collections.ErrNotFound).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no err unmarshalling, no err from GetAttestationRequestsByHeight, voteExt oracle att length > length request, reject
	attReq := bridgetypes.AttestationRequests{
		Requests: []*bridgetypes.AttestationRequest{
			{
				Snapshot: []byte("snapshot"),
			},
		},
	}
	bridgeVoteExt.OracleAttestations = append(bridgeVoteExt.OracleAttestations, app.OracleAttestation{
		Attestation: []byte("attestation2"),
		Snapshot:    []byte("snapshot2"),
	})
	require.Equal(len(bridgeVoteExt.OracleAttestations), 2)
	bridgeVoteExtBz, err = json.Marshal(bridgeVoteExt)
	require.NoError(err)
	req = &abci.RequestVerifyVoteExtension{
		VoteExtension: bridgeVoteExtBz,
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no err unmarshalling, no err from GetAttestationRequestsByHeight, voteExt oracle att length == length request, initial sig too big, reject
	bridgeVoteExt.InitialSignature.SignatureA = make([]byte, 100000)
	bridgeVoteExt.OracleAttestations = []app.OracleAttestation{
		oracleAtt,
	}
	bridgeVoteExtBz, err = json.Marshal(bridgeVoteExt)
	require.NoError(err)
	req = &abci.RequestVerifyVoteExtension{
		VoteExtension: bridgeVoteExtBz,
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no err unmarshalling, no err from GetAttestationRequestsByHeight, voteExt oracle att length == length request, initial sig good, valset sig too big, reject
	bridgeVoteExt.ValsetSignature.Signature = make([]byte, 100000)
	bridgeVoteExt.InitialSignature = app.InitialSignature{
		SignatureA: []byte("signature"),
		SignatureB: []byte("signature"),
	}
	bridgeVoteExtBz, err = json.Marshal(bridgeVoteExt)
	require.NoError(err)
	req = &abci.RequestVerifyVoteExtension{
		VoteExtension: bridgeVoteExtBz,
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no errs unmarshalling, no err from GetAttestationRequestsByHeight, voteExt oracle att length == length request, initial sig good, valset sig good, accept
	bridgeVoteExt.ValsetSignature.Signature = []byte("signature")
	bridgeVoteExtBz, err = json.Marshal(bridgeVoteExt)
	require.NoError(err)
	req = &abci.RequestVerifyVoteExtension{
		VoteExtension: bridgeVoteExtBz,
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_ACCEPT)
}

// create a keyring in the test, sotre in temp directort, modify voteexthandler
// or read youre own computer
// maybe even pass in a keyring instead of reading it

func (s *VoteExtensionTestSuite) TestSignMessage() {
	require := s.Require()
	h := s.handler

	// Initial keyring state
	keys, err := s.kr.List()
	require.NoError(err)
	fmt.Println("Initial keys in keyring: ", keys)

	res, err := h.SignMessage([]byte("msg"))
	require.Error(err)
	require.Nil(res)

	// create key
	// cmd := exec.Command("layerd", "keys", "add", "key0", "--keyring-backend", "test", "--keyring-dir", s.tempDir)
	// output, err := cmd.CombinedOutput()
	// require.NoError(err)
	// require.NotNil(output)
	// fmt.Println(string(output))

	// cmd = exec.Command("layerd", "keys", "list", "--keyring-backend", "test", "--keyring-dir", s.tempDir)
	// output, err = cmd.CombinedOutput()
	// require.NoError(err)
	// require.NotNil(output)
	// fmt.Println(string(output))

	// Log the keyring state after adding the key
	// keys, err = s.kr.List()
	// require.NoError(err)
	// fmt.Println("Keys in keyring after adding key: ", keys)

	// viper.Set("key-name", "key0")
	// res, err = h.SignMessage([]byte("msg"))
	// require.NoError(err)
	// require.NotNil(res)

	// kr := s.kr
	// uid := "testOne"
	// // encryptPassphrase := "this passphrase has been used for all test vectors"
	// // armor := "-----BEGIN TENDERMINT PRIVATE KEY-----\nkdf: bcrypt\nsalt: 6BC5D5187F9DF241E1A1243EECFF9C17\ntype: secp256k1\n\nGDPpPfrSVZloiwufbal19fmd75QeiqwToZ949SwmnxxM03qL75xXVf3tTD/BrF4l\nFs14HuhwntDBM2xgZvymTBk2edHlEI20Phv6oC0=\n=/zZh\n-----END TENDERMINT PRIVATE KEY-----"
	// // err := kr.ImportPrivKey(uid, armor, encryptPassphrase)
	// // require.NoError(err)
	// record, mn, err := kr.NewMnemonic(uid, keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	// require.NoError(err)
	// require.NotNil(mn)
	// require.NotNil(record)
	// record2, err := kr.NewAccount(uid, mn, "", "", hd.Secp256k1)
	// require.NoError(err)
	// require.NotNil(record2)

	// // List keys after adding
	// keys, err := kr.List()
	// require.NoError(err)
	// fmt.Println("Keys in keyring after cmd: ", keys)
	// s.T().TempDir()

	// Create a temporary directory for the keyring
	// tempDir, err := os.MkdirTemp("", "keyring-test")
	// require.NoError(err)
	// defer os.RemoveAll(tempDir)

	// viper.Set("keyring-backend", "test")
	// viper.Set("keyring-dir", tempDir)
	// viper.Set("key-name", "key5")

	// kr, err := s.handler.InitKeyring()
	// require.NoError(err)
	// require.NotNil(kr)
	// fmt.Println("Initialized keyring: ", kr)

	// // Add key using command
	// cmd := exec.Command("layerd", "keys", "add", "key5", "--keyring-dir", tempDir)
	// output, err := cmd.CombinedOutput()
	// require.NoError(err)
	// require.NotNil(output)
	// fmt.Println("Output from adding key: ", string(output))

	// // List keys after adding
	// keys, err := kr.List()
	// require.NoError(err)
	// require.NotNil(keys)
	// fmt.Println("Keys in keyring: ", keys)
}

func (s *VoteExtensionTestSuite) TestGetKeyring() {
	require := s.Require()
	h := s.handler

	kr, err := h.GetKeyring()
	require.NoError(err)
	require.NotNil(kr)
}

func (s *VoteExtensionTestSuite) TestGetOperatorAddress() {
	// require := s.Require()
	// h := s.handler
	// kr := s.kr

	// viper.Set("key-name", "key20")

	// addr, err := h.GetOperatorAddress()
	// require.NoError(err)
	// require.NotNil(addr)
}

// use test case struct

func (s *VoteExtensionTestSuite) TestExtendVoteHandler() {
	require := s.Require()
	h := s.handler
	bk := s.bridgeKeeper
	ctx := s.ctx.WithBlockHeight(3)
	patches := gomonkey.NewPatches()

	type testCase struct {
		name             string
		setupMocks       func()
		request          *abci.RequestExtendVote
		expectedError    error
		validateResponse func(*abci.ResponseExtendVote)
	}

	testCases := []testCase{
		{
			name:          "GetOperatorAddress error",
			setupMocks:    func() {},
			request:       &abci.RequestExtendVote{},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err on SignInitialMessage",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err on GetAttestationRequestsByHeight",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				patches.ApplyMethod(h, "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
					return []byte("signatureA"), []byte("signatureB"), nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(nil, errors.New("error"))
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err signing checkpoint",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				evmAddr := common.BytesToAddress([]byte("evmAddr"))
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(evmAddr.Bytes(), nil)
				attReq := bridgetypes.AttestationRequests{
					Requests: []*bridgetypes.AttestationRequest{
						{
							Snapshot: []byte("snapshot"),
						},
					},
				}
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(&attReq, nil)
				patches.ApplyMethod(h, "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return []byte("signedMsg"), nil
				})
				// bk.On("GetLatestCheckpointIndex", ctx).Return(nil, errors.New("error"))
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		// {
		// 	name: "GetAttestationRequestsByHeight error",
		// 	setupMocks: func() {
		// 		oppAddr := sample.AccAddress()
		// 		patches := gomonkey.NewPatches()
		// 		defer patches.Reset()
		// 		patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
		// 			return oppAddr, nil
		// 		})
		// 		patches.ApplyMethod(h, "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
		// 			return []byte("signatureA"), []byte("signatureB"), nil
		// 		})
		// 		bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, errors.New("error")).Once()
		// 		bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(nil, errors.New("error")).Once()
		// 	},
		// 	expectedError: nil,
		// 	validateResponse: func(resp *abci.ResponseExtendVote) {
		// 		require.NotNil(resp)
		// 	},
		// },
		// {
		// 	name: "Signing error",
		// 	setupMocks: func() {
		// 		oppAddr := sample.AccAddress()
		// 		attReq := bridgetypes.AttestationRequests{
		// 			Requests: []*bridgetypes.AttestationRequest{
		// 				{
		// 					Snapshot: []byte("snapshot"),
		// 				},
		// 			},
		// 		}
		// 		patches := gomonkey.NewPatches()
		// 		defer patches.Reset()
		// 		patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
		// 			return oppAddr, nil
		// 		})
		// 		patches.ApplyMethod(h, "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
		// 			return []byte("signatureA"), []byte("signatureB"), nil
		// 		})
		// 		bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, errors.New("error")).Once()
		// 		bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(&attReq, nil).Once()
		// 	},
		// 	expectedError: nil,
		// 	validateResponse: func(resp *abci.ResponseExtendVote) {
		// 		require.NotNil(resp)
		// 	},
		// },
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setupMocks()
			defer patches.Reset()
			req := &abci.RequestExtendVote{}
			resp, err := h.ExtendVoteHandler(ctx, req)
			if tc.expectedError != nil {
				require.Error(err)
				require.Equal(tc.expectedError, err)
			} else {
				require.NoError(err)
			}
			tc.validateResponse(resp)
		})
	}
}
