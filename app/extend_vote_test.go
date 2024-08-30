package app_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/agiledragon/gomonkey/v2"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/testutil/sample"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/api/cosmos/crypto/secp256k1"
	"cosmossdk.io/collections"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
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
	kr            *mocks.Keyring
	cdc           codec.Codec
}

func (s *VoteExtensionTestSuite) SetupTest() {
	registry := codectypes.NewInterfaceRegistry()
	s.cdc = codec.NewProtoCodec(registry)
	s.oracleKeeper = mocks.NewOracleKeeper(s.T())
	s.bridgeKeeper = mocks.NewBridgeKeeper(s.T())
	s.stakingKeeper = mocks.NewStakingKeeper(s.T())

	// create new vote ext handler
	s.ctx = testutils.CreateTestContext(s.T())
	s.handler = app.NewVoteExtHandler(
		log.NewNopLogger(),
		s.cdc,
		s.oracleKeeper,
		s.bridgeKeeper,
	)
}

func (s *VoteExtensionTestSuite) TearDownTest() {
	s.oracleKeeper.AssertExpectations(s.T())
	s.bridgeKeeper.AssertExpectations(s.T())
	s.stakingKeeper.AssertExpectations(s.T())
	viper.Reset()
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

	s.TearDownTest()
}

func (s *VoteExtensionTestSuite) TestExtendVoteHandler() {
	require := s.Require()
	h := s.handler
	bk := s.bridgeKeeper
	ctx := s.ctx.WithBlockHeight(3)
	patches := gomonkey.NewPatches()

	type testCase struct {
		name             string
		setupMocks       func()
		expectedError    error
		validateResponse func(*abci.ResponseExtendVote)
	}

	testCases := []testCase{
		{
			name:          "GetOperatorAddress error",
			setupMocks:    func() {},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err on SignInitialMessage",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound).Once()
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
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				patches.ApplyMethod(reflect.TypeOf(h), "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
					return []byte("signatureA"), []byte("signatureB"), nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound).Once()
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return((*bridgetypes.AttestationRequests)(nil), errors.New("error!")).Once()
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
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(evmAddr.Bytes(), nil).Once()
				attReq := bridgetypes.AttestationRequests{
					Requests: []*bridgetypes.AttestationRequest{
						{
							Snapshot: []byte("snapshot"),
						},
					},
				}
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(&attReq, nil).Once()
				patches.ApplyMethod(reflect.TypeOf(h), "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return []byte("signedMsg"), nil
				})
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("error")).Once()
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "no errors",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				evmAddr := common.BytesToAddress([]byte("evmAddr"))
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(evmAddr.Bytes(), nil).Once()
				attReq := bridgetypes.AttestationRequests{
					Requests: []*bridgetypes.AttestationRequest{
						{
							Snapshot: []byte("snapshot"),
						},
					},
				}
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(&attReq, nil).Once()
				patches.ApplyMethod(reflect.TypeOf(h), "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return []byte("signedMsg"), nil
				})
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				checkpointTimestamp := bridgetypes.CheckpointTimestamp{
					Timestamp: 1,
				}
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(checkpointTimestamp, nil).Once()
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(1)).Return(true, int64(1), nil).Once()
				patches.ApplyMethod(reflect.TypeOf(h), "EncodeAndSignMessage", func(_ *app.VoteExtHandler, checkpoint string) ([]byte, error) {
					return []byte("signedCheckpoint"), nil
				})
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			defer patches.Reset()
			tc.setupMocks()
			req := &abci.RequestExtendVote{}
			resp, err := h.ExtendVoteHandler(ctx, req)
			if tc.expectedError != nil {
				require.Error(err)
				require.Equal(tc.expectedError, err)
			} else {
				require.NoError(err)
			}
			tc.validateResponse(resp)
			s.TearDownTest()
		})
	}
}

func (s *VoteExtensionTestSuite) TestGetKeyring() {
	require := s.Require()
	h := s.handler

	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", s.T().TempDir())
	kr, err := h.GetKeyring()
	require.NoError(err)
	require.NotNil(kr)

	s.TearDownTest()
}

func (s *VoteExtensionTestSuite) TestGetOperatorAddress() {
	require := s.Require()
	// h := s.handler
	patches := gomonkey.NewPatches()

	type testCase struct {
		name             string
		setupMocks       func(h *app.VoteExtHandler) *gomonkey.Patches
		expectedError    bool
		expectedErrorMsg string
	}

	testCases := []testCase{
		{
			name: "err getting keyring",
			setupMocks: func(h *app.VoteExtHandler) *gomonkey.Patches {
				s.kr = nil
				return nil
			},
			expectedError:    true,
			expectedErrorMsg: "failed to get keyring:",
		},
		{
			name: "err getting keyname",
			setupMocks: func(h *app.VoteExtHandler) *gomonkey.Patches {
				s.kr = mocks.NewKeyring(s.T())
				tempDir := s.T().TempDir()
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				return patches
			},
			expectedError:    true,
			expectedErrorMsg: "key name not found, please set --key-name flag",
		},
		{
			name: "empty keyring",
			setupMocks: func(h *app.VoteExtHandler) *gomonkey.Patches {
				tempDir := s.T().TempDir()
				viper.Set("key-name", "testkey")
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				mockKr := mocks.NewKeyring(s.T())
				mockKr.On("List").Return([]*keyring.Record{}, nil).Once()

				patches := gomonkey.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
						return mockKr, nil
					})
				return patches
			},
			expectedError:    true,
			expectedErrorMsg: "no keys found in keyring",
		},
		{
			name: "kr.Key error",
			setupMocks: func(h *app.VoteExtHandler) *gomonkey.Patches {
				tempDir := s.T().TempDir()
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				viper.Set("key-name", "testkey")
				pubKey := &secp256k1.PubKey{Key: []byte("pubkey")}
				anyPubKey, err := codectypes.NewAnyWithValue(pubKey)
				require.NoError(err)
				mockKr := mocks.NewKeyring(s.T())
				mockKr.On("List").Return([]*keyring.Record{
					{Name: "testkey", PubKey: anyPubKey},
				}, nil).Once()
				mockKr.On("Key", "testkey").Return(nil, errors.New("error!")).Once()

				patches := gomonkey.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
						return mockKr, nil
					})
				return patches
			},
			expectedError:    true,
			expectedErrorMsg: "failed to get operator key:",
		},
		{
			name: "success",
			setupMocks: func(h *app.VoteExtHandler) *gomonkey.Patches {
				tempDir := s.T().TempDir()
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				viper.Set("key-name", "testkey")
				priv := hd.Secp256k1.Generate()([]byte("test"))
				// Create a public key
				pubKey := priv.PubKey()
				// Create an AnyPubKey
				anyPubKey, err := codectypes.NewAnyWithValue(pubKey)
				require.NoError(err)
				// Create a local item
				localItem, err := keyring.NewLocalRecord(
					"testkey",
					priv,
					pubKey,
				)
				require.NoError(err)
				mockKr := mocks.NewKeyring(s.T())
				mockKr.On("List").Return([]*keyring.Record{
					{Name: "testkey", PubKey: anyPubKey},
				}, nil).Once()

				mockKr.On("Key", "testkey").Return(&keyring.Record{
					Name:   "testkey",
					PubKey: anyPubKey,
					Item:   localItem.Item,
				}, nil).Once()
				fmt.Println("anyPubKey: ", anyPubKey)
				patches := gomonkey.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
						return mockKr, nil
					})
				return patches
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			handler := app.NewVoteExtHandler(
				log.NewNopLogger(),
				s.cdc,
				s.oracleKeeper,
				s.bridgeKeeper,
			)
			defer patches.Reset()
			defer viper.Reset()
			tc.setupMocks(handler)
			resp, err := handler.GetOperatorAddress()
			if tc.expectedError {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedErrorMsg)
			} else {
				require.NoError(err)
				require.NotEmpty(resp)
			}
			s.TearDownTest()
		})
	}
}

func (s *VoteExtensionTestSuite) TestSignInitialMessage() {
	require := s.Require()
	h := s.handler

	patches := gomonkey.NewPatches()
	patches.ApplyMethod(reflect.TypeOf(h), "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
		return []byte("signedMsg"), nil
	})

	sigA, sigB, err := h.SignInitialMessage()
	require.NoError(err)
	require.NotEmpty(sigA)
	require.NotEmpty(sigB)
}

func (s *VoteExtensionTestSuite) TestCheckAndSignValidatorCheckpoint() {
	require := s.Require()
	h := s.handler
	bk := s.bridgeKeeper
	ctx := s.ctx.WithBlockHeight(2)

	testCases := []struct {
		name              string
		setupMocks        func()
		expectedSig       []byte
		expectedTimestamp uint64
		expectedError     error
	}{
		{
			name: "Validator already signed",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				s.mockGetOperatorAddress(oppAddr, nil)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(true, int64(1), nil).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     nil,
		},
		{
			name: "Error getting latest checkpoint index",
			setupMocks: func() {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("index error")).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("index error"),
		},
		{
			name: "Error getting validator timestamp",
			setupMocks: func() {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{}, errors.New("timestamp error")).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("timestamp error"),
		},
		{
			name: "Error getting operator address",
			setupMocks: func() {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				s.mockGetOperatorAddress("", errors.New("operator address error"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("operator address error"),
		},
		{
			name: "Error checking if validator signed",
			setupMocks: func() {
				oppAddr := sample.AccAddress()
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				s.mockGetOperatorAddress(oppAddr, nil)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(false, int64(0), errors.New("sign check error")).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("sign check error"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setupMocks()
			sig, timestamp, err := h.CheckAndSignValidatorCheckpoint(ctx)
			if tc.expectedError != nil {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(err)
			}
			require.Equal(tc.expectedSig, sig)
			require.Equal(tc.expectedTimestamp, timestamp)
		})
	}

	s.TearDownTest()
}

func (s *VoteExtensionTestSuite) mockGetOperatorAddress(address string, err error) {
	patches := gomonkey.NewPatches()
	patches.ApplyMethod(reflect.TypeOf(s.handler), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
		return address, err
	})
	s.T().Cleanup(patches.Reset)
}

// func (s *VoteExtensionTestSuite) mockSignMessage(signature []byte, err error) {
// 	patches := gomonkey.NewPatches()
// 	patches.ApplyMethod(reflect.TypeOf(s.handler), "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
// 		return signature, err
// 	})
// 	s.T().Cleanup(patches.Reset)
// }

func (s *VoteExtensionTestSuite) TestEncodeAndSignMessage() {
	require := s.Require()
	h := s.handler

	tests := []struct {
		name              string
		checkpointString  string
		mockSignature     []byte
		mockError         error
		expectedSignature []byte
		expectedError     string
	}{
		{
			name:              "Valid checkpoint",
			checkpointString:  "0123456789abcdef",
			mockSignature:     []byte("mocksignature"),
			mockError:         nil,
			expectedSignature: []byte("mocksignature"),
			expectedError:     "",
		},
		{
			name:              "Invalid hex string",
			checkpointString:  "invalid hex",
			mockSignature:     nil,
			mockError:         nil,
			expectedSignature: nil,
			expectedError:     "encoding/hex: invalid byte",
		},
		{
			name:              "SignMessage error",
			checkpointString:  "0123456789abcdef",
			mockSignature:     nil,
			mockError:         errors.New("signing error"),
			expectedSignature: nil,
			expectedError:     "signing error",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Mock the SignMessage method
			patches := gomonkey.ApplyMethod(reflect.TypeOf(h), "SignMessage",
				func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return tt.mockSignature, tt.mockError
				})
			defer patches.Reset()

			// Call the function
			signature, err := h.EncodeAndSignMessage(tt.checkpointString)

			// Assert the results
			if tt.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tt.expectedError)
			} else {
				require.NoError(err)
			}
			require.Equal(tt.expectedSignature, signature)
		})
	}
}

func (s *VoteExtensionTestSuite) TestGetValidatorIndexInValset() {
	require := s.Require()
	h := s.handler
	ctx := s.ctx

	testCases := []struct {
		name          string
		evmAddr       []byte
		valset        *bridgetypes.BridgeValidatorSet
		expectedIndex int
		expectedError error
	}{
		{
			name:    "Validator found at index 0",
			evmAddr: []byte("evmAddr1"),
			valset: &bridgetypes.BridgeValidatorSet{
				BridgeValidatorSet: []*bridgetypes.BridgeValidator{
					{EthereumAddress: []byte("evmAddr1"), Power: 1},
					{EthereumAddress: []byte("evmAddr2"), Power: 2},
				},
			},
			expectedIndex: 0,
			expectedError: nil,
		},
		{
			name:    "Validator found at index 1",
			evmAddr: []byte("evmAddr2"),
			valset: &bridgetypes.BridgeValidatorSet{
				BridgeValidatorSet: []*bridgetypes.BridgeValidator{
					{EthereumAddress: []byte("evmAddr1"), Power: 1},
					{EthereumAddress: []byte("evmAddr2"), Power: 2},
				},
			},
			expectedIndex: 1,
			expectedError: nil,
		},
		{
			name:    "Validator not found",
			evmAddr: []byte("evmAddr3"),
			valset: &bridgetypes.BridgeValidatorSet{
				BridgeValidatorSet: []*bridgetypes.BridgeValidator{
					{EthereumAddress: []byte("evmAddr1"), Power: 1},
					{EthereumAddress: []byte("evmAddr2"), Power: 2},
				},
			},
			expectedIndex: -1,
			expectedError: errors.New("validator not found in valset"),
		},
		{
			name:    "Empty valset",
			evmAddr: []byte("evmAddr1"),
			valset: &bridgetypes.BridgeValidatorSet{
				BridgeValidatorSet: []*bridgetypes.BridgeValidator{},
			},
			expectedIndex: -1,
			expectedError: errors.New("validator not found in valset"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			index, err := h.GetValidatorIndexInValset(ctx, tc.evmAddr, tc.valset)

			if tc.expectedError != nil {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(err)
			}

			require.Equal(tc.expectedIndex, index)
		})
	}
}

// func (s *VoteExtensionTestSuite) TestSignMessage() {
// 	require := s.Require()
// 	h := s.handler

// 	testCases := []struct {
// 		name          string
// 		message       []byte
// 		expectedError bool
// 		setup         func() *gomonkey.Patches
// 	}{
// 		{
// 			name:          "GetKeyring error",
// 			message:       []byte("msg"),
// 			expectedError: true,
// 			setup: func() *gomonkey.Patches {
// 				patches := gomonkey.NewPatches()
// 				patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
// 					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
// 						return nil, errors.New("GetKeyring error!")
// 					})
// 				return patches
// 			},
// 		},
// {
// 	name:          "key-name not set",
// 	message:       []byte("msg"),
// 	expectedError: true,
// 	setup: func() *gomonkey.Patches {
// 		patches := gomonkey.NewPatches()
// 		mockKr := mocks.NewKeyring(s.T())
// 		patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
// 			func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
// 				return mockKr, nil
// 			})
// 		viper.Set("key-name", "")
// 		return patches
// 	},
// },
// {
// 	name:          "kr.Sign error",
// 	message:       []byte("msg"),
// 	expectedError: true,
// 	setup: func() {
// 		mockKr := mocks.NewKeyring(s.T())
// 		patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
// 			func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
// 				return mockKr, nil
// 			})
// 		viper.Set("key-name", "testkey")
// 		mockKr.On("Sign", "testkey", []byte("msg"), mock.Anything).Return(nil, errors.New("sign error")).Once()
// 	},
// },
// {
// 	name:          "success",
// 	message:       []byte("msg"),
// 	expectedError: false,
// 	setup: func() {
// 		mockKr := mocks.NewKeyring(s.T())
// 		patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
// 			func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
// 				return mockKr, nil
// 			})
// 		viper.Set("key-name", "testkey")
// 		mockKr.On("Sign", "testkey", []byte("msg"), mock.Anything).Return([]byte("signature"), nil).Once()
// 	},
// },
// 	}

// 	for _, tc := range testCases {
// 		s.Run(tc.name, func() {
// 			patches := tc.setup()
// 			defer viper.Reset()

// 			res, err := h.SignMessage(tc.message)
// 			if tc.expectedError {
// 				fmt.Println("tc.name: ", tc.name, ", err: ", err)
// 				require.Error(err)
// 				require.Nil(res)
// 			} else {
// 				require.NoError(err)
// 				require.NotNil(res)
// 			}
// 			patches.Reset()
// 		})
// 	}

// 	s.TearDownTest()
// }
