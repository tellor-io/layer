package app_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

	"cosmossdk.io/api/cosmos/crypto/secp256k1"
	"cosmossdk.io/api/cosmos/crypto/secp256k1"
	"cosmossdk.io/collections"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type VoteExtensionTestSuite struct {
	suite.Suite
	ctx sdk.Context
	kr  *mocks.Keyring
	cdc codec.Codec
}

func (s *VoteExtensionTestSuite) SetupTest() {
	registry := codectypes.NewInterfaceRegistry()
	s.cdc = codec.NewProtoCodec(registry)

	s.ctx = testutils.CreateTestContext(s.T())
	viper.Reset()
}

func (s *VoteExtensionTestSuite) CreateHandlerAndMocks() (*app.VoteExtHandler, *mocks.OracleKeeper, *mocks.BridgeKeeper, *mocks.StakingKeeper) {
	oracleKeeper := mocks.NewOracleKeeper(s.T())
	bridgeKeeper := mocks.NewBridgeKeeper(s.T())
	stakingKeeper := mocks.NewStakingKeeper(s.T())

	handler := app.NewVoteExtHandler(
		log.NewNopLogger(),
		s.cdc,
		oracleKeeper,
		bridgeKeeper,
	)

	return handler, oracleKeeper, bridgeKeeper, stakingKeeper
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

// TODO: turn into test case array
func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler() {
	require := s.Require()
	h, _, bk, _ := s.CreateHandlerAndMocks()

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
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(nil, collections.ErrNotFound).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// no err unmarshalling, no err from GetAttestationRequestsByHeight, voteExt oracle att length > length request, reject
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
	// bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil).Once()
	// bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil).Once()
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_ACCEPT)
}

func (s *VoteExtensionTestSuite) TestExtendVoteHandler() {
	require := s.Require()
	ctx := s.ctx.WithBlockHeight(3)

	oppAddr := sample.AccAddress()
	evmAddr := common.BytesToAddress([]byte("evmAddr"))

	type testCase struct {
		name             string
		setupMocks       func(bk *mocks.BridgeKeeper, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches)
		expectedError    error
		validateResponse func(*abci.ResponseExtendVote)
	}

	testCases := []testCase{
		{
			name: "err on SignInitialMessage",
			setupMocks: func(bk *mocks.BridgeKeeper, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				patches.ApplyMethod(reflect.TypeOf(h), "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
					return nil, nil, errors.New("error!")
				})
				return bk, patches
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err on GetAttestationRequestsByHeight",
			setupMocks: func(bk *mocks.BridgeKeeper, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				patches.ApplyMethod(reflect.TypeOf(h), "SignInitialMessage", func(_ *app.VoteExtHandler) ([]byte, []byte, error) {
					return []byte("signatureA"), []byte("signatureB"), nil
				})
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return((*bridgetypes.AttestationRequests)(nil), errors.New("error!"))
				return bk, patches
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err signing checkpoint",
			setupMocks: func(bk *mocks.BridgeKeeper, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
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
				patches.ApplyMethod(reflect.TypeOf(h), "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return []byte("signedMsg"), nil
				})
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("error"))
				return bk, patches
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "no errors",
			// order:
			// 1. h.GetOperatorAddress()
			// 2. h.bk.GetEVMAddressByOperator()
			// 3. h.bk.GetAttestationRequestsByHeight()
			// 4. h.SignMessage()
			// 5. h.CheckAndSignValidatorCheckpoint()
			// 5a. h.bk.GetLatestCheckpointIndex()
			// 5b. h.bk.GetValidatorTimestampByIdxFromStorage()
			// 5c. h.GetOperatorAddress
			// 5d. h.bk.GetValidatorDidSignCheckpoint()
			setupMocks: func(bk *mocks.BridgeKeeper, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				// 1.
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				// 2.
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(evmAddr.Bytes(), nil)
				attReq := bridgetypes.AttestationRequests{
					Requests: []*bridgetypes.AttestationRequest{
						{
							Snapshot: []byte("snapshot"),
						},
					},
				}
				// 3.
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(&attReq, nil)
				// 4.
				patches.ApplyMethod(reflect.TypeOf(h), "SignMessage",
					func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
						return []byte("signedMsg"), nil
					})
				// 5a.
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				checkpointTimestamp := bridgetypes.CheckpointTimestamp{
					Timestamp: 1,
				}
				// 5b.
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(checkpointTimestamp, nil)
				// 5c.
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				// 5d.
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(1)).Return(true, int64(1), nil)

				return bk, patches
			},
			expectedError: nil,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			patches := gomonkey.NewPatches()
			s.T().Cleanup(func() {
				patches.Reset()
			})
			h, _, bk, _ := s.CreateHandlerAndMocks()
			if tc.setupMocks != nil {
				bk, patches = tc.setupMocks(bk, h, patches)
				require.NotNil(bk)
				require.NotNil(patches)
			}
			req := &abci.RequestExtendVote{}
			resp, err := h.ExtendVoteHandler(ctx, req)
			if tc.expectedError != nil {
				require.Error(err)
				require.Equal(tc.expectedError, err)
			} else {
				require.NoError(err)
			}
			tc.validateResponse(resp)
			bk.AssertExpectations(s.T())
			fmt.Println(tc.name, " finished!!")
		})
	}
}

func (s *VoteExtensionTestSuite) TestGetKeyring() {
	require := s.Require()
	h, _, _, _ := s.CreateHandlerAndMocks()

	viper.Set("keyring-backend", "test")
	viper.Set("keyring-dir", s.T().TempDir())
	kr, err := h.GetKeyring()
	require.NoError(err)
	require.NotNil(kr)

	s.T().Cleanup(func() {
		viper.Reset()
	})
}

func (s *VoteExtensionTestSuite) TestGetOperatorAddress() {
	require := s.Require()

	type testCase struct {
		name             string
		setupMocks       func(h *app.VoteExtHandler, patches *gomonkey.Patches)
		expectedError    bool
		expectedErrorMsg string
	}

	testCases := []testCase{
		{
			name: "err getting keyname",
			setupMocks: func(h *app.VoteExtHandler, patches *gomonkey.Patches) {
				s.kr = mocks.NewKeyring(s.T())
				tempDir := s.T().TempDir()
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				viper.Set("key-name", "")
			},
			expectedError:    true,
			expectedErrorMsg: "key name not found, please set --key-name flag",
		},
		{
			name: "empty keyring",
			setupMocks: func(h *app.VoteExtHandler, patches *gomonkey.Patches) {
				tempDir := s.T().TempDir()
				viper.Set("key-name", "testkey")
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				mockKr := mocks.NewKeyring(s.T())
				mockKr.On("List").Return([]*keyring.Record{}, nil)

				patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
						return mockKr, nil
					})
			},
			expectedError:    true,
			expectedErrorMsg: "no keys found in keyring",
		},
		{
			name: "kr.Key error",
			setupMocks: func(h *app.VoteExtHandler, patches *gomonkey.Patches) {
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
				}, nil)
				mockKr.On("Key", "testkey").Return(nil, errors.New("error!"))

				patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
						return mockKr, nil
					})
			},
			expectedError:    true,
			expectedErrorMsg: "failed to get operator key:",
		},
		{
			name: "success",
			setupMocks: func(h *app.VoteExtHandler, patches *gomonkey.Patches) {
				tempDir := s.T().TempDir()
				viper.Set("keyring-backend", "test")
				viper.Set("keyring-dir", tempDir)
				viper.Set("key-name", "testkey")
				priv := hd.Secp256k1.Generate()([]byte("test"))
				pubKey := priv.PubKey()
				anyPubKey, err := codectypes.NewAnyWithValue(pubKey)
				require.NoError(err)
				localItem, err := keyring.NewLocalRecord(
					"testkey",
					priv,
					pubKey,
				)
				require.NoError(err)
				mockKr := mocks.NewKeyring(s.T())
				mockKr.On("List").Return([]*keyring.Record{
					{Name: "testkey", PubKey: anyPubKey},
				}, nil)

				mockKr.On("Key", "testkey").Return(&keyring.Record{
					Name:   "testkey",
					PubKey: anyPubKey,
					Item:   localItem.Item,
				}, nil)
				patches.ApplyMethod(reflect.TypeOf(h), "GetKeyring",
					func(_ *app.VoteExtHandler) (keyring.Keyring, error) {
						return mockKr, nil
					})
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			patches := gomonkey.NewPatches()
			h, _, _, _ := s.CreateHandlerAndMocks()
			s.T().Cleanup(func() {
				patches.Reset()
				viper.Reset()
			})
			if tc.setupMocks != nil {
				tc.setupMocks(h, patches)
			}
			resp, err := h.GetOperatorAddress()
			if tc.expectedError {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedErrorMsg)
			} else {
				require.NoError(err)
				require.NotEmpty(resp)
			}
		})
	}
}

func (s *VoteExtensionTestSuite) TestSignInitialMessage() {
	require := s.Require()
	h, _, _, _ := s.CreateHandlerAndMocks()

	patches := gomonkey.NewPatches()
	patches.ApplyMethod(reflect.TypeOf(h), "SignMessage", func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
		return []byte("signedMsg"), nil
	})

	sigA, sigB, err := h.SignInitialMessage()
	require.NoError(err)
	require.NotEmpty(sigA)
	require.NotEmpty(sigB)

	s.T().Cleanup(func() {
		patches.Reset()
	})
}

func (s *VoteExtensionTestSuite) TestCheckAndSignValidatorCheckpoint() {
	require := s.Require()
	ctx := s.ctx.WithBlockHeight(2)

	oppAddr := sample.AccAddress()

	testCases := []struct {
		name              string
		setupMocks        func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, patches *gomonkey.Patches)
		expectedSig       []byte
		expectedTimestamp uint64
		expectedError     error
	}{
		{
			name: "Validator already signed",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(true, int64(1), nil).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     nil,
		},
		{
			name: "Error getting latest checkpoint index",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("index error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("index error"),
		},
		{
			name: "Error getting validator timestamp",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil)
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{}, errors.New("timestamp error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("timestamp error"),
		},
		{
			name: "Error checking if validator signed",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil)
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil)
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				fmt.Println("oppAddr in setupMocks", oppAddr)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(false, int64(0), errors.New("sig check error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("sig check error!"),
		},
		{
			name: "No errors",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				patches.ApplyMethod(reflect.TypeOf(h), "GetOperatorAddress", func(_ *app.VoteExtHandler) (string, error) {
					return oppAddr, nil
				})
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(false, int64(1), nil).Once()
				bk.On("GetValidatorCheckpointParamsFromStorage", ctx, uint64(10)).Return(bridgetypes.ValidatorCheckpointParams{
					Checkpoint: []byte("checkpoint"),
				}, nil).Once()
				patches.ApplyMethod(reflect.TypeOf(h), "EncodeAndSignMessage", func(_ *app.VoteExtHandler, msg string) ([]byte, error) {
					return []byte("signedMsg"), nil
				})
			},
			expectedSig:       []byte("signedMsg"),
			expectedTimestamp: 10,
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			fmt.Println("NAME: ", tc.name)
			patches := gomonkey.NewPatches()
			fmt.Println("Patches: ", patches)
			s.T().Cleanup(func() {
				patches.Reset()
			})
			fmt.Println("Patches: ", patches)

			h, _, bk, _ := s.CreateHandlerAndMocks()
			if tc.setupMocks != nil {
				tc.setupMocks(h, bk, patches)
			}
			fmt.Println("Patches: ", patches)
			sig, timestamp, err := h.CheckAndSignValidatorCheckpoint(ctx)
			if tc.expectedError != nil {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(err)
			}
			require.Equal(tc.expectedSig, sig)
			require.Equal(tc.expectedTimestamp, timestamp)
			bk.AssertExpectations(s.T())
			fmt.Println(tc.name, " finished!!")
		})
	}
}

func (s *VoteExtensionTestSuite) TestEncodeAndSignMessage() {
	require := s.Require()

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
			h, _, _, _ := s.CreateHandlerAndMocks()
			patches := gomonkey.NewPatches()
			patches.ApplyMethod(reflect.TypeOf(h), "SignMessage",
				func(_ *app.VoteExtHandler, msg []byte) ([]byte, error) {
					return tt.mockSignature, tt.mockError
				})

			signature, err := h.EncodeAndSignMessage(tt.checkpointString)

			if tt.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tt.expectedError)
			} else {
				require.NoError(err)
			}
			require.Equal(tt.expectedSignature, signature)
			s.T().Cleanup(func() {
				patches.Reset()
			})
		})
	}
}

func (s *VoteExtensionTestSuite) TestGetValidatorIndexInValset() {
	require := s.Require()
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
			h, _, _, _ := s.CreateHandlerAndMocks()
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
