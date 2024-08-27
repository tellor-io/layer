package app_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
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

type MockKeyring struct {
	mock.Mock
}

func (m *MockKeyring) List() ([]*keyring.Record, error) {
	args := m.Called()
	return args.Get(0).([]*keyring.Record), args.Error(1)
}

func (s *VoteExtensionTestSuite) SetupTest() {
	require := s.Require()

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

	s.tempDir = s.T().TempDir()
	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", s.tempDir)
	var err error
	s.kr, err = s.handler.InitKeyring()
	require.NoError(err)
	require.NotNil(s.kr)
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
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
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
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				patches.ApplyMethod(h, "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
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
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
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
				patches.ApplyMethod(h, "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
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
				patches.ApplyMethod(h, "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
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
				patches.ApplyMethod(h, "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return []byte("signedMsg"), nil
				})
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				checkpointTimestamp := bridgetypes.CheckpointTimestamp{
					Timestamp: 1,
				}
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(checkpointTimestamp, nil).Once()
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(1)).Return(true, int64(1), nil).Once()
				patches.ApplyMethod(h, "EncodeAndSignMessage", func(_ *app.VoteExtHandler, checkpoint string) ([]byte, error) {
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
			defer patches.Reset()
		})
	}
}

func (s *VoteExtensionTestSuite) TestSignMessage() {
	require := s.Require()
	h := s.handler
	kr := s.kr

	// Initial keyring state
	keys, err := s.kr.List()
	require.NoError(err)
	require.Len(keys, 0)

	testCases := []struct {
		name          string
		message       []byte
		expectedError bool
		setup         func()
	}{
		{
			name:          "Empty keyring",
			message:       []byte("msg"),
			expectedError: true,
		},
		// Add more test cases here
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.setup != nil {
				tc.setup()
			}
			res, err := h.SignMessage(tc.message)
			if tc.expectedError {
				require.Error(err)
				require.Nil(res)
			} else {
				require.NoError(err)
				require.NotNil(res)
			}
			defer func() {
				viper.Set("key-name", "")
			}()
		})
	}
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
	patches := gomonkey.NewPatches()

	type testCase struct {
		name             string
		setupMocks       func()
		expectedError    bool
		expectedErrorMsg string
	}

	testCases := []testCase{
		{
			name: "err getting keyring",
			setupMocks: func() {
				patches.ApplyMethod(h, "GetKeyring", func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
					return nil, errors.New("error!")
				})
			},
			expectedError:    true,
			expectedErrorMsg: "failed to get keyring: error!",
		},
		{
			name: "err getting keyname",
			setupMocks: func() {
				viper.Set("key-name", "")
			},
			expectedError:    true,
			expectedErrorMsg: "key name not found, please set --key-name flag",
		},
		{
			name: "keyring list error",
			setupMocks: func() {
				viper.Set("key-name", "testkey")
				patches.ApplyMethod(reflect.TypeOf(s.kr), "List", func(_ keyring.Keyring) ([]keyring.Record, error) {
					return nil, errors.New("error!")
				})
			},
			expectedError:    true,
			expectedErrorMsg: "failed to list keys: error!",
		},
		{
			name: "keyring list success",
			setupMocks: func() {
				viper.Set("key-name", "testkey")
				patches.ApplyMethod(reflect.TypeOf(s.kr), "List", func(_ keyring.Keyring) ([]keyring.Record, error) {
					return []keyring.Record{}, nil
				})
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setupMocks()
			defer patches.Reset()
			resp, err := h.GetOperatorAddress()
			if tc.expectedError {
				require.Error(err)
				require.Equal(tc.expectedErrorMsg, err.Error())
			} else {
				require.NoError(err)
				require.NotEmpty(resp)
			}
		})
	}
}
