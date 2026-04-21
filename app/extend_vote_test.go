package app_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
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
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type VoteExtensionTestSuite struct {
	suite.Suite
	ctx sdk.Context
	cdc codec.Codec
}

func (s *VoteExtensionTestSuite) SetupTest() {
	registry := codectypes.NewInterfaceRegistry()
	s.cdc = codec.NewProtoCodec(registry)

	s.ctx = testutils.CreateTestContext(s.T())
}

func (s *VoteExtensionTestSuite) CreateHandlerAndMocks() (*app.VoteExtHandler, *mocks.OracleKeeper, *mocks.BridgeKeeper, *mocks.StakingKeeper, *mocks.VoteExtensionSigner) {
	oracleKeeper := mocks.NewOracleKeeper(s.T())
	bridgeKeeper := mocks.NewBridgeKeeper(s.T())
	stakingKeeper := mocks.NewStakingKeeper(s.T())
	signer := mocks.NewVoteExtensionSigner(s.T())

	handler := app.NewVoteExtHandler(
		log.NewNopLogger(),
		s.cdc,
		oracleKeeper,
		bridgeKeeper,
		signer,
	)

	return handler, oracleKeeper, bridgeKeeper, stakingKeeper, signer
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

// TODO: turn into test case array
func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler() {
	require := s.Require()
	h, _, bk, _, _ := s.CreateHandlerAndMocks()

	res, err := h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{})
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_REJECT)

	// err unmarshalling, validator has evm address, val has EVM addr, reject
	req := &abci.RequestVerifyVoteExtension{
		ValidatorAddress: []byte("operatorIn"),
	}

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
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_ACCEPT)

	bk.AssertExpectations(s.T())
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler_RejectsUnknownFields() {
	require := s.Require()
	h, _, bk, _, _ := s.CreateHandlerAndMocks()
	s.ctx = s.ctx.WithBlockHeight(3)

	attReq := bridgetypes.AttestationRequests{
		Requests: []*bridgetypes.AttestationRequest{
			{Snapshot: make([]byte, 32)},
		},
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil)

	validVE := &app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{Snapshot: make([]byte, 32), Attestation: make([]byte, 65)},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: make([]byte, 65),
			SignatureB: make([]byte, 65),
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: make([]byte, 65),
			Timestamp: 1,
		},
	}

	// valid VE should be accepted
	bz, err := json.Marshal(validVE)
	require.NoError(err)
	res, err := h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{VoteExtension: bz})
	require.NoError(err)
	require.Equal(abci.ResponseVerifyVoteExtension_ACCEPT, res.Status)

	// inject an unknown JSON field -- should be rejected by DisallowUnknownFields
	injected := make([]byte, len(bz))
	copy(injected, bz)
	// replace closing '}' with ',"_pad":"junk"}'
	injected = append(injected[:len(injected)-1], []byte(`,"_pad":"junk"}`)...)
	res, err = h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{VoteExtension: injected})
	require.NoError(err)
	require.Equal(abci.ResponseVerifyVoteExtension_REJECT, res.Status)
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler_RejectsTrailingJSONData() {
	require := s.Require()
	h, _, _, _, _ := s.CreateHandlerAndMocks()
	s.ctx = s.ctx.WithBlockHeight(3)

	validVE := &app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{Snapshot: make([]byte, 32), Attestation: make([]byte, 65)},
		},
		InitialSignature: app.InitialSignature{
			SignatureA: make([]byte, 65),
			SignatureB: make([]byte, 65),
		},
		ValsetSignature: app.BridgeValsetSignature{
			Signature: make([]byte, 65),
			Timestamp: 1,
		},
	}

	bz, err := json.Marshal(validVE)
	require.NoError(err)

	// Append a second top-level JSON value; should be rejected.
	bzWithTrailingData := append(append([]byte{}, bz...), []byte(` {}`)...)
	res, err := h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{VoteExtension: bzWithTrailingData})
	require.NoError(err)
	require.Equal(abci.ResponseVerifyVoteExtension_REJECT, res.Status)
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler_RejectsOversizedRawVE() {
	require := s.Require()
	h, _, _, _, _ := s.CreateHandlerAndMocks()

	// create a payload larger than maxVoteExtensionSize (512KB)
	oversized := make([]byte, 512*1024+1)
	// fill with valid-looking JSON start so it's clearly over the limit before parsing
	copy(oversized, []byte(`{"OracleAttestations":[]}`))
	res, err := h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{VoteExtension: oversized})
	require.NoError(err)
	require.Equal(abci.ResponseVerifyVoteExtension_REJECT, res.Status)
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler_RejectsOversizedAttestationFields() {
	require := s.Require()
	h, _, bk, _, _ := s.CreateHandlerAndMocks()
	s.ctx = s.ctx.WithBlockHeight(3)

	attReq := bridgetypes.AttestationRequests{
		Requests: []*bridgetypes.AttestationRequest{
			{Snapshot: make([]byte, 32)},
		},
	}
	bk.On("GetAttestationRequestsByHeight", s.ctx, uint64(2)).Return(&attReq, nil)

	// oversized snapshot (>32 bytes) should be rejected
	veOversizedSnapshot := &app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{Snapshot: make([]byte, 100), Attestation: make([]byte, 65)},
		},
		InitialSignature: app.InitialSignature{},
		ValsetSignature:  app.BridgeValsetSignature{},
	}
	bz, err := json.Marshal(veOversizedSnapshot)
	require.NoError(err)
	res, err := h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{VoteExtension: bz})
	require.NoError(err)
	require.Equal(abci.ResponseVerifyVoteExtension_REJECT, res.Status)

	// oversized attestation sig (>65 bytes) should be rejected
	veOversizedAttSig := &app.BridgeVoteExtension{
		OracleAttestations: []app.OracleAttestation{
			{Snapshot: make([]byte, 32), Attestation: make([]byte, 100)},
		},
		InitialSignature: app.InitialSignature{},
		ValsetSignature:  app.BridgeValsetSignature{},
	}
	bz, err = json.Marshal(veOversizedAttSig)
	require.NoError(err)
	res, err = h.VerifyVoteExtensionHandler(s.ctx, &abci.RequestVerifyVoteExtension{VoteExtension: bz})
	require.NoError(err)
	require.Equal(abci.ResponseVerifyVoteExtension_REJECT, res.Status)
}

func (s *VoteExtensionTestSuite) TestExtendVoteHandler() {
	require := s.Require()
	ctx := s.ctx.WithBlockHeight(3)

	oppAddr := sample.AccAddress()
	evmAddr := common.BytesToAddress([]byte("evmAddr"))

	type testCase struct {
		name             string
		setupMocks       func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches)
		expectedPanic    bool
		validateResponse func(*abci.ResponseExtendVote)
	}

	testCases := []testCase{
		{
			name: "err on SignInitialMessage",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				patches.ApplyMethod(reflect.TypeOf(h), "SignInitialMessage", func(_ *app.VoteExtHandler, operatorAddress string) ([]byte, []byte, error) {
					return nil, nil, errors.New("error!")
				})
				return bk, patches
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err on GetOperatorAddress",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return("", errors.New("error!"))
				return bk, patches
			},
			expectedPanic: true,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				// This won't be called due to the panic
			},
		},
		{
			name: "err on GetAttestationRequestsByHeight",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				patches.ApplyMethod(reflect.TypeOf(h), "SignInitialMessage", func(_ *app.VoteExtHandler, operatorAddress string) ([]byte, []byte, error) {
					return []byte("signatureA"), []byte("signatureB"), nil
				})
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return((*bridgetypes.AttestationRequests)(nil), errors.New("error!"))
				return bk, patches
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "no EVM address, real SignInitialMessage succeeds, no attestations",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				// Let real SignInitialMessage run — mock signer.Sign with exact expected hashes
				hashA := sha256.Sum256([]byte(fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", oppAddr)))
				hashB := sha256.Sum256([]byte(fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", oppAddr)))
				signer.On("Sign", mock.Anything, hashA[:]).Return([]byte("sigA"), nil).Once()
				signer.On("Sign", mock.Anything, hashB[:]).Return([]byte("sigB"), nil).Once()
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(nil, collections.ErrNotFound)
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("no checkpoint"))
				return bk, patches
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
				// Verify the initial signatures were included in the vote extension
				var voteExt app.BridgeVoteExtension
				require.NoError(json.Unmarshal(resp.VoteExtension, &voteExt))
				require.Equal([]byte("sigA"), voteExt.InitialSignature.SignatureA)
				require.Equal([]byte("sigB"), voteExt.InitialSignature.SignatureB)
			},
		},
		{
			name: "err signing checkpoint",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(evmAddr.Bytes(), nil)
				attReq := bridgetypes.AttestationRequests{
					Requests: []*bridgetypes.AttestationRequest{
						{
							Snapshot: []byte("snapshot"),
						},
					},
				}
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(&attReq, nil)
				signer.On("Sign", mock.Anything, []byte("snapshot")).Return([]byte("signedMsg"), nil)
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("error"))
				return bk, patches
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "no errors",
			// order:
			// 1. h.signer.GetOperatorAddress()
			// 2. h.bk.GetEVMAddressByOperator()
			// 3. h.bk.GetAttestationRequestsByHeight()
			// 4. h.signer.Sign()
			// 5. h.CheckAndSignValidatorCheckpoint()
			// 5a. h.bk.GetLatestCheckpointIndex()
			// 5b. h.bk.GetValidatorTimestampByIdxFromStorage()
			// 5c. h.signer.GetOperatorAddress()
			// 5d. h.bk.GetValidatorDidSignCheckpoint()
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) (*mocks.BridgeKeeper, *gomonkey.Patches) {
				// 1 + 5c.
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
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
				signer.On("Sign", mock.Anything, []byte("snapshot")).Return([]byte("signedMsg"), nil)
				// 5a.
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				checkpointTimestamp := bridgetypes.CheckpointTimestamp{
					Timestamp: 1,
				}
				// 5b.
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(checkpointTimestamp, nil)
				// 5d.
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(1)).Return(true, int64(1), nil)

				return bk, patches
			},
			expectedPanic: false,
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
			h, _, bk, _, signer := s.CreateHandlerAndMocks()
			if tc.setupMocks != nil {
				bk, patches = tc.setupMocks(bk, signer, h, patches)
				require.NotNil(bk)
				require.NotNil(patches)
			}

			// mock forceProcessTermination to prevent actual process termination
			patches.ApplyMethod(reflect.TypeOf(h), "ForceProcessTermination",
				func(_ *app.VoteExtHandler, format string, args ...interface{}) {
					// instead of terminating, panic with the error message
					panic(fmt.Sprintf(format, args...))
				})

			req := &abci.RequestExtendVote{}

			if tc.expectedPanic {
				require.Panics(func() {
					_, _ = h.ExtendVoteHandler(ctx, req)
				})
			} else {
				resp, err := h.ExtendVoteHandler(ctx, req)
				require.NoError(err)
				tc.validateResponse(resp)
			}

			bk.AssertExpectations(s.T())
		})
	}
}

func (s *VoteExtensionTestSuite) TestSignInitialMessage() {
	require := s.Require()

	operatorAddr := "operatorAddr1"
	expectedHashA := sha256.Sum256([]byte(fmt.Sprintf("TellorLayer: Initial bridge signature A for operator %s", operatorAddr)))
	expectedHashB := sha256.Sum256([]byte(fmt.Sprintf("TellorLayer: Initial bridge signature B for operator %s", operatorAddr)))

	testCases := []struct {
		name          string
		setupSigner   func(signer *mocks.VoteExtensionSigner)
		expectedSigA  []byte
		expectedSigB  []byte
		expectedError string
	}{
		{
			name: "success",
			setupSigner: func(signer *mocks.VoteExtensionSigner) {
				signer.On("Sign", mock.Anything, expectedHashA[:]).Return([]byte("signedMsgA"), nil).Once()
				signer.On("Sign", mock.Anything, expectedHashB[:]).Return([]byte("signedMsgB"), nil).Once()
			},
			expectedSigA: []byte("signedMsgA"),
			expectedSigB: []byte("signedMsgB"),
		},
		{
			name: "error signing message A",
			setupSigner: func(signer *mocks.VoteExtensionSigner) {
				signer.On("Sign", mock.Anything, expectedHashA[:]).Return(nil, errors.New("sign A failed")).Once()
			},
			expectedError: "failed to sign message A",
		},
		{
			name: "error signing message B",
			setupSigner: func(signer *mocks.VoteExtensionSigner) {
				signer.On("Sign", mock.Anything, expectedHashA[:]).Return([]byte("signedMsgA"), nil).Once()
				signer.On("Sign", mock.Anything, expectedHashB[:]).Return(nil, errors.New("sign B failed")).Once()
			},
			expectedError: "failed to sign message B",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			h, _, _, _, signer := s.CreateHandlerAndMocks()
			tc.setupSigner(signer)

			sigA, sigB, err := h.SignInitialMessage(operatorAddr)
			if tc.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError)
			} else {
				require.NoError(err)
				require.Equal(tc.expectedSigA, sigA)
				require.Equal(tc.expectedSigB, sigB)
			}
		})
	}
}

func (s *VoteExtensionTestSuite) TestCheckAndSignValidatorCheckpoint() {
	require := s.Require()
	ctx := s.ctx.WithBlockHeight(2)

	oppAddr := sample.AccAddress()

	testCases := []struct {
		name              string
		setupMocks        func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, patches *gomonkey.Patches)
		expectedSig       []byte
		expectedTimestamp uint64
		expectedError     error
	}{
		{
			name: "Validator already signed",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(true, int64(1), nil).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     nil,
		},
		{
			name: "Error getting latest checkpoint index",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("index error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("index error"),
		},
		{
			name: "Error getting validator timestamp",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil)
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{}, errors.New("timestamp error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("timestamp error"),
		},
		{
			name: "Error checking if validator signed",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil)
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil)
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(false, int64(0), errors.New("sig check error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("sig check error!"),
		},
		{
			name: "No errors",
			setupMocks: func(h *app.VoteExtHandler, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, patches *gomonkey.Patches) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
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
			patches := gomonkey.NewPatches()
			patches.Reset()
			h, _, bk, _, signer := s.CreateHandlerAndMocks()
			if tc.setupMocks != nil {
				tc.setupMocks(h, bk, signer, patches)
			}
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
			s.T().Cleanup(func() {
				patches.Reset()
			})
		})
	}
}

func (s *VoteExtensionTestSuite) TestEncodeAndSignMessage() {
	require := s.Require()

	expectedBytes, _ := hex.DecodeString("0123456789abcdef")

	tests := []struct {
		name               string
		checkpointString   string
		setupSigner        func(signer *mocks.VoteExtensionSigner)
		expectedSignature  []byte
		expectedError      string
		signShouldBeCalled bool
	}{
		{
			name:             "Valid checkpoint",
			checkpointString: "0123456789abcdef",
			setupSigner: func(signer *mocks.VoteExtensionSigner) {
				signer.On("Sign", mock.Anything, expectedBytes).Return([]byte("signedMsg"), nil).Once()
			},
			expectedSignature:  []byte("signedMsg"),
			signShouldBeCalled: true,
		},
		{
			name:             "Invalid hex string",
			checkpointString: "invalid hex",
			setupSigner: func(signer *mocks.VoteExtensionSigner) {
				// Sign should not be called when hex decoding fails
			},
			expectedSignature:  nil,
			expectedError:      "encoding/hex: invalid byte",
			signShouldBeCalled: false,
		},
		{
			name:             "Sign error",
			checkpointString: "0123456789abcdef",
			setupSigner: func(signer *mocks.VoteExtensionSigner) {
				signer.On("Sign", mock.Anything, expectedBytes).Return(nil, errors.New("signing error")).Once()
			},
			expectedSignature:  nil,
			expectedError:      "signing error",
			signShouldBeCalled: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			h, _, _, _, signer := s.CreateHandlerAndMocks()
			tt.setupSigner(signer)

			signature, err := h.EncodeAndSignMessage(tt.checkpointString)

			if tt.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tt.expectedError)
			} else {
				require.NoError(err)
			}
			require.Equal(tt.expectedSignature, signature)

			if !tt.signShouldBeCalled {
				signer.AssertNotCalled(s.T(), "Sign", mock.Anything, mock.Anything)
			}
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
			h, _, _, _, _ := s.CreateHandlerAndMocks()
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
