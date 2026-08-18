package app_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	signerv1 "github.com/tellor-io/bridge-remote-signer/api/gen/signer/v1"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"
	"github.com/tellor-io/layer/testutil/sample"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

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
	viper.Reset()
}

func (s *VoteExtensionTestSuite) CreateHandlerAndMocks() (*app.VoteExtHandler, *mocks.OracleKeeper, *mocks.BridgeKeeper, *mocks.VoteExtensionSigner) {
	oracleKeeper := mocks.NewOracleKeeper(s.T())
	bridgeKeeper := mocks.NewBridgeKeeper(s.T())
	signer := mocks.NewVoteExtensionSigner(s.T())

	handler := app.NewVoteExtHandler(
		log.NewNopLogger(),
		s.cdc,
		oracleKeeper,
		bridgeKeeper,
		signer,
	)

	return handler, oracleKeeper, bridgeKeeper, signer
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

// TODO: turn into test case array
func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler() {
	require := s.Require()
	h, _, bk, _ := s.CreateHandlerAndMocks()

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
	res, err = h.VerifyVoteExtensionHandler(s.ctx, req)
	require.NoError(err)
	require.Equal(res.Status, abci.ResponseVerifyVoteExtension_ACCEPT)

	bk.AssertExpectations(s.T())
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtHandler_RejectsUnknownFields() {
	require := s.Require()
	h, _, bk, _ := s.CreateHandlerAndMocks()
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
	h, _, _, _ := s.CreateHandlerAndMocks()
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
	h, _, _, _ := s.CreateHandlerAndMocks()

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
	h, _, bk, _ := s.CreateHandlerAndMocks()
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
	snapshotData := bridgetypes.AttestationSnapshotData{
		ValidatorCheckpoint:    []byte("valCheckpoint"),
		AttestationTimestamp:   4,
		PrevReportTimestamp:    1,
		NextReportTimestamp:    3,
		QueryId:                []byte("queryId"),
		Timestamp:              2,
		LastConsensusTimestamp: 2,
	}
	aggregate := oracletypes.Aggregate{
		AggregateValue: "abcd1234",
		AggregatePower: 10,
	}

	type testCase struct {
		name             string
		setupMocks       func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches)
		expectedPanic    bool
		validateResponse func(*abci.ResponseExtendVote)
	}

	testCases := []testCase{
		{
			name: "err on SignInitial",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				signer.On("SignInitial", mock.Anything).Return(nil, nil, errors.New("error!")).Once()
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "err on GetOperatorAddress",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return("", errors.New("error!"))
			},
			expectedPanic: true,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				// This won't be called due to the panic
			},
		},
		{
			name: "err on GetAttestationRequestsByHeight",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				signer.On("SignInitial", mock.Anything).Return([]byte("signature"), []byte("signature"), nil).Once()
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return((*bridgetypes.AttestationRequests)(nil), errors.New("error!"))
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
			},
		},
		{
			name: "no EVM address, SignInitial succeeds, no attestations",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) {
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetEVMAddressByOperator", ctx, oppAddr).Return(nil, collections.ErrNotFound)
				signer.On("SignInitial", mock.Anything).Return([]byte("sigA"), []byte("sigB"), nil).Once()
				bk.On("GetAttestationRequestsByHeight", ctx, uint64(2)).Return(nil, collections.ErrNotFound)
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("no checkpoint"))
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
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) {
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
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, []byte("snapshot")).Return(snapshotData, nil)
				ok.On("GetAggregateByTimestamp", ctx, snapshotData.QueryId, snapshotData.Timestamp).Return(aggregate, nil)
				signer.On("SignOracleAttestation", mock.Anything, mock.Anything).Return([]byte("signedMsg"), nil)
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("error"))
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
			// 4. h.SignSnapshot() -> h.signer.SignOracleAttestation()
			// 5. h.CheckAndSignValidatorCheckpoint()
			// 5a. h.bk.GetLatestCheckpointIndex()
			// 5b. h.bk.GetValidatorTimestampByIdxFromStorage()
			// 5c. h.signer.GetOperatorAddress()
			// 5d. h.bk.GetValidatorDidSignCheckpoint()
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner, h *app.VoteExtHandler, patches *gomonkey.Patches) {
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
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, []byte("snapshot")).Return(snapshotData, nil)
				ok.On("GetAggregateByTimestamp", ctx, snapshotData.QueryId, snapshotData.Timestamp).Return(aggregate, nil)
				expectedValue, err := hex.DecodeString(aggregate.AggregateValue)
				require.NoError(err)
				signer.On("SignOracleAttestation", mock.Anything, mock.MatchedBy(func(req *signerv1.SignOracleAttestationRequest) bool {
					return string(req.ExpectedSnapshot) == "snapshot" &&
						string(req.QueryId) == string(snapshotData.QueryId) &&
						string(req.Value) == string(expectedValue) &&
						req.AggregatePower == aggregate.AggregatePower &&
						req.AttestationTimestamp == snapshotData.AttestationTimestamp
				})).Return([]byte("signedMsg"), nil)
				// 5a.
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				checkpointTimestamp := bridgetypes.CheckpointTimestamp{
					Timestamp: 1,
				}
				// 5b.
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(checkpointTimestamp, nil)
				// 5d.
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(1)).Return(true, int64(1), nil)
			},
			expectedPanic: false,
			validateResponse: func(resp *abci.ResponseExtendVote) {
				require.NotNil(resp)
				var voteExt app.BridgeVoteExtension
				require.NoError(json.Unmarshal(resp.VoteExtension, &voteExt))
				require.Len(voteExt.OracleAttestations, 1)
				require.Equal([]byte("signedMsg"), voteExt.OracleAttestations[0].Attestation)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			patches := gomonkey.NewPatches()
			s.T().Cleanup(func() {
				patches.Reset()
			})
			h, ok, bk, signer := s.CreateHandlerAndMocks()
			if tc.setupMocks != nil {
				tc.setupMocks(ok, bk, signer, h, patches)
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

func (s *VoteExtensionTestSuite) TestSignSnapshot() {
	require := s.Require()
	ctx := s.ctx

	snapshot := []byte("snapshot")
	snapshotData := bridgetypes.AttestationSnapshotData{
		ValidatorCheckpoint:    []byte("valCheckpoint"),
		AttestationTimestamp:   4,
		PrevReportTimestamp:    1,
		NextReportTimestamp:    3,
		QueryId:                []byte("queryId"),
		Timestamp:              2,
		LastConsensusTimestamp: 2,
	}
	aggregate := oracletypes.Aggregate{
		AggregateValue: "0xabcd1234",
		AggregatePower: 10,
	}

	testCases := []struct {
		name          string
		setupMocks    func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner)
		expectedSig   []byte
		expectedError string
	}{
		{
			name: "success, structured request carries the snapshot inputs",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, snapshot).Return(snapshotData, nil)
				ok.On("GetAggregateByTimestamp", ctx, snapshotData.QueryId, snapshotData.Timestamp).Return(aggregate, nil)
				expectedValue, err := hex.DecodeString("abcd1234") // 0x prefix stripped
				require.NoError(err)
				signer.On("SignOracleAttestation", mock.Anything, mock.MatchedBy(func(req *signerv1.SignOracleAttestationRequest) bool {
					return string(req.ExpectedSnapshot) == string(snapshot) &&
						string(req.QueryId) == string(snapshotData.QueryId) &&
						string(req.Value) == string(expectedValue) &&
						req.Timestamp == snapshotData.Timestamp &&
						req.AggregatePower == aggregate.AggregatePower &&
						req.PreviousTimestamp == snapshotData.PrevReportTimestamp &&
						req.NextTimestamp == snapshotData.NextReportTimestamp &&
						string(req.ValsetCheckpoint) == string(snapshotData.ValidatorCheckpoint) &&
						req.AttestationTimestamp == snapshotData.AttestationTimestamp &&
						req.LastConsensusTimestamp == snapshotData.LastConsensusTimestamp
				})).Return([]byte("signedMsg"), nil)
			},
			expectedSig: []byte("signedMsg"),
		},
		{
			name: "error getting snapshot data",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, snapshot).Return(bridgetypes.AttestationSnapshotData{}, errors.New("snapshot data error"))
			},
			expectedError: "snapshot data error",
		},
		{
			name: "error getting aggregate",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, snapshot).Return(snapshotData, nil)
				ok.On("GetAggregateByTimestamp", ctx, snapshotData.QueryId, snapshotData.Timestamp).Return(oracletypes.Aggregate{}, errors.New("aggregate error"))
			},
			expectedError: "aggregate error",
		},
		{
			name: "invalid hex aggregate value",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, snapshot).Return(snapshotData, nil)
				ok.On("GetAggregateByTimestamp", ctx, snapshotData.QueryId, snapshotData.Timestamp).Return(oracletypes.Aggregate{AggregateValue: "not hex"}, nil)
			},
			expectedError: "encoding/hex: invalid byte",
		},
		{
			name: "signer error",
			setupMocks: func(ok *mocks.OracleKeeper, bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetAttestationSnapshotDataBySnapshot", ctx, snapshot).Return(snapshotData, nil)
				ok.On("GetAggregateByTimestamp", ctx, snapshotData.QueryId, snapshotData.Timestamp).Return(aggregate, nil)
				signer.On("SignOracleAttestation", mock.Anything, mock.Anything).Return(nil, errors.New("signer refused"))
			},
			expectedError: "signer refused",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			h, ok, bk, signer := s.CreateHandlerAndMocks()
			tc.setupMocks(ok, bk, signer)

			sig, err := h.SignSnapshot(ctx, snapshot)
			if tc.expectedError != "" {
				require.Error(err)
				require.Contains(err.Error(), tc.expectedError)
			} else {
				require.NoError(err)
				require.Equal(tc.expectedSig, sig)
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
		setupMocks        func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner)
		expectedSig       []byte
		expectedTimestamp uint64
		expectedError     error
	}{
		{
			name: "Validator already signed",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
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
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(0), errors.New("index error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("index error"),
		},
		{
			name: "Error getting validator timestamp",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil)
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{}, errors.New("timestamp error!"))
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("timestamp error"),
		},
		{
			name: "Error checking if validator signed",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
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
			name: "Signer refuses to sign checkpoint",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(false, int64(1), nil).Once()
				bk.On("GetValidatorCheckpointParamsFromStorage", ctx, uint64(10)).Return(bridgetypes.ValidatorCheckpointParams{
					Checkpoint: []byte("checkpoint"),
				}, nil).Once()
				bk.On("GetValsetCheckpointDomainSeparator", ctx).Return([]byte("domainSep"), nil).Once()
				bk.On("GetBridgeValsetByTimestamp", ctx, uint64(10)).Return(&bridgetypes.BridgeValidatorSet{}, nil).Once()
				signer.On("SignCheckpoint", mock.Anything, mock.Anything).Return(nil, errors.New("checkpoint mismatch")).Once()
			},
			expectedSig:       nil,
			expectedTimestamp: 0,
			expectedError:     errors.New("checkpoint mismatch"),
		},
		{
			name: "No errors",
			setupMocks: func(bk *mocks.BridgeKeeper, signer *mocks.VoteExtensionSigner) {
				bk.On("GetLatestCheckpointIndex", ctx).Return(uint64(1), nil).Once()
				bk.On("GetValidatorTimestampByIdxFromStorage", ctx, uint64(1)).Return(bridgetypes.CheckpointTimestamp{
					Timestamp: 10,
				}, nil).Once()
				signer.On("GetOperatorAddress", mock.Anything).Return(oppAddr, nil)
				bk.On("GetValidatorDidSignCheckpoint", ctx, oppAddr, uint64(10)).Return(false, int64(1), nil).Once()
				bk.On("GetValidatorCheckpointParamsFromStorage", ctx, uint64(10)).Return(bridgetypes.ValidatorCheckpointParams{
					Checkpoint:     []byte("checkpoint"),
					ValsetHash:     []byte("valsetHash"),
					Timestamp:      10,
					PowerThreshold: 66,
					BlockHeight:    5,
				}, nil).Once()
				bk.On("GetValsetCheckpointDomainSeparator", ctx).Return([]byte("domainSep"), nil).Once()
				bk.On("GetBridgeValsetByTimestamp", ctx, uint64(10)).Return(&bridgetypes.BridgeValidatorSet{
					BridgeValidatorSet: []*bridgetypes.BridgeValidator{
						{EthereumAddress: []byte("evmAddr1"), Power: 100},
					},
				}, nil).Once()
				signer.On("SignCheckpoint", mock.Anything, mock.MatchedBy(func(req *signerv1.SignBridgeCheckpointRequest) bool {
					return string(req.ExpectedCheckpoint) == "checkpoint" &&
						string(req.DomainSeparator) == "domainSep" &&
						string(req.ValidatorSetHash) == "valsetHash" &&
						req.PowerThreshold == 66 &&
						req.ValidatorTimestamp == 10 &&
						req.BlockHeight == 5 &&
						req.CheckpointIndex == 1 &&
						len(req.ValidatorSet) == 1 &&
						req.ValidatorSet[0].Power == 100
				})).Return([]byte("signedMsg"), nil).Once()
			},
			expectedSig:       []byte("signedMsg"),
			expectedTimestamp: 10,
			expectedError:     nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			h, _, bk, signer := s.CreateHandlerAndMocks()
			if tc.setupMocks != nil {
				tc.setupMocks(bk, signer)
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
