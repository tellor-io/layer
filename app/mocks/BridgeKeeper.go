// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/ethereum/go-ethereum/common"

	log "cosmossdk.io/log"

	mock "github.com/stretchr/testify/mock"

	types "github.com/tellor-io/layer/x/bridge/types"
)

// BridgeKeeper is an autogenerated mock type for the BridgeKeeper type
type BridgeKeeper struct {
	mock.Mock
}

// EVMAddressFromSignatures provides a mock function with given fields: ctx, sigA, sigB, operatorAddress
func (_m *BridgeKeeper) EVMAddressFromSignatures(ctx context.Context, sigA []byte, sigB []byte, operatorAddress string) (common.Address, error) {
	ret := _m.Called(ctx, sigA, sigB, operatorAddress)

	var r0 common.Address
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, []byte, string) (common.Address, error)); ok {
		return rf(ctx, sigA, sigB, operatorAddress)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, []byte, string) common.Address); ok {
		r0 = rf(ctx, sigA, sigB, operatorAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, []byte, string) error); ok {
		r1 = rf(ctx, sigA, sigB, operatorAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAttestationRequestsByHeight provides a mock function with given fields: ctx, height
func (_m *BridgeKeeper) GetAttestationRequestsByHeight(ctx context.Context, height uint64) (*types.AttestationRequests, error) {
	ret := _m.Called(ctx, height)

	var r0 *types.AttestationRequests
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (*types.AttestationRequests, error)); ok {
		return rf(ctx, height)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) *types.AttestationRequests); ok {
		r0 = rf(ctx, height)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.AttestationRequests)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, height)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBridgeValsetByTimestamp provides a mock function with given fields: ctx, timestamp
func (_m *BridgeKeeper) GetBridgeValsetByTimestamp(ctx context.Context, timestamp uint64) (*types.BridgeValidatorSet, error) {
	ret := _m.Called(ctx, timestamp)

	var r0 *types.BridgeValidatorSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (*types.BridgeValidatorSet, error)); ok {
		return rf(ctx, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) *types.BridgeValidatorSet); ok {
		r0 = rf(ctx, timestamp)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BridgeValidatorSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEVMAddressByOperator provides a mock function with given fields: ctx, operatorAddress
func (_m *BridgeKeeper) GetEVMAddressByOperator(ctx context.Context, operatorAddress string) ([]byte, error) {
	ret := _m.Called(ctx, operatorAddress)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]byte, error)); ok {
		return rf(ctx, operatorAddress)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []byte); ok {
		r0 = rf(ctx, operatorAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, operatorAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestCheckpointIndex provides a mock function with given fields: ctx
func (_m *BridgeKeeper) GetLatestCheckpointIndex(ctx context.Context) (uint64, error) {
	ret := _m.Called(ctx)

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (uint64, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) uint64); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetValidatorCheckpointFromStorage provides a mock function with given fields: ctx
func (_m *BridgeKeeper) GetValidatorCheckpointFromStorage(ctx context.Context) (*types.ValidatorCheckpoint, error) {
	ret := _m.Called(ctx)

	var r0 *types.ValidatorCheckpoint
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*types.ValidatorCheckpoint, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *types.ValidatorCheckpoint); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ValidatorCheckpoint)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetValidatorCheckpointParamsFromStorage provides a mock function with given fields: ctx, timestamp
func (_m *BridgeKeeper) GetValidatorCheckpointParamsFromStorage(ctx context.Context, timestamp uint64) (types.ValidatorCheckpointParams, error) {
	ret := _m.Called(ctx, timestamp)

	var r0 types.ValidatorCheckpointParams
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (types.ValidatorCheckpointParams, error)); ok {
		return rf(ctx, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) types.ValidatorCheckpointParams); ok {
		r0 = rf(ctx, timestamp)
	} else {
		r0 = ret.Get(0).(types.ValidatorCheckpointParams)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetValidatorDidSignCheckpoint provides a mock function with given fields: ctx, operatorAddr, checkpointTimestamp
func (_m *BridgeKeeper) GetValidatorDidSignCheckpoint(ctx context.Context, operatorAddr string, checkpointTimestamp uint64) (bool, int64, error) {
	ret := _m.Called(ctx, operatorAddr, checkpointTimestamp)

	var r0 bool
	var r1 int64
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, uint64) (bool, int64, error)); ok {
		return rf(ctx, operatorAddr, checkpointTimestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, uint64) bool); ok {
		r0 = rf(ctx, operatorAddr, checkpointTimestamp)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, uint64) int64); ok {
		r1 = rf(ctx, operatorAddr, checkpointTimestamp)
	} else {
		r1 = ret.Get(1).(int64)
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, uint64) error); ok {
		r2 = rf(ctx, operatorAddr, checkpointTimestamp)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetValidatorSetSignaturesFromStorage provides a mock function with given fields: ctx, timestamp
func (_m *BridgeKeeper) GetValidatorSetSignaturesFromStorage(ctx context.Context, timestamp uint64) (*types.BridgeValsetSignatures, error) {
	ret := _m.Called(ctx, timestamp)

	var r0 *types.BridgeValsetSignatures
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (*types.BridgeValsetSignatures, error)); ok {
		return rf(ctx, timestamp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) *types.BridgeValsetSignatures); ok {
		r0 = rf(ctx, timestamp)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BridgeValsetSignatures)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, timestamp)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetValidatorTimestampByIdxFromStorage provides a mock function with given fields: ctx, checkpointIdx
func (_m *BridgeKeeper) GetValidatorTimestampByIdxFromStorage(ctx context.Context, checkpointIdx uint64) (types.CheckpointTimestamp, error) {
	ret := _m.Called(ctx, checkpointIdx)

	var r0 types.CheckpointTimestamp
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (types.CheckpointTimestamp, error)); ok {
		return rf(ctx, checkpointIdx)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) types.CheckpointTimestamp); ok {
		r0 = rf(ctx, checkpointIdx)
	} else {
		r0 = ret.Get(0).(types.CheckpointTimestamp)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, checkpointIdx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Logger provides a mock function with given fields: ctx
func (_m *BridgeKeeper) Logger(ctx context.Context) log.Logger {
	ret := _m.Called(ctx)

	var r0 log.Logger
	if rf, ok := ret.Get(0).(func(context.Context) log.Logger); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(log.Logger)
		}
	}

	return r0
}

// SetBridgeValsetSignature provides a mock function with given fields: ctx, operatorAddress, timestamp, signature
func (_m *BridgeKeeper) SetBridgeValsetSignature(ctx context.Context, operatorAddress string, timestamp uint64, signature string) error {
	ret := _m.Called(ctx, operatorAddress, timestamp, signature)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, uint64, string) error); ok {
		r0 = rf(ctx, operatorAddress, timestamp, signature)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetEVMAddressByOperator provides a mock function with given fields: ctx, operatorAddr, evmAddr
func (_m *BridgeKeeper) SetEVMAddressByOperator(ctx context.Context, operatorAddr string, evmAddr []byte) error {
	ret := _m.Called(ctx, operatorAddr, evmAddr)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte) error); ok {
		r0 = rf(ctx, operatorAddr, evmAddr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetOracleAttestation provides a mock function with given fields: ctx, operatorAddress, snapshot, sig
func (_m *BridgeKeeper) SetOracleAttestation(ctx context.Context, operatorAddress string, snapshot []byte, sig []byte) error {
	ret := _m.Called(ctx, operatorAddress, snapshot, sig)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte, []byte) error); ok {
		r0 = rf(ctx, operatorAddress, snapshot, sig)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewBridgeKeeper interface {
	mock.TestingT
	Cleanup(func())
}

// NewBridgeKeeper creates a new instance of BridgeKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBridgeKeeper(t mockConstructorTestingTNewBridgeKeeper) *BridgeKeeper {
	mock := &BridgeKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
