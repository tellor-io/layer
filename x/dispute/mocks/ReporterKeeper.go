// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	math "cosmossdk.io/math"

	mock "github.com/stretchr/testify/mock"

	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	types "github.com/cosmos/cosmos-sdk/types"
)

// ReporterKeeper is an autogenerated mock type for the ReporterKeeper type
type ReporterKeeper struct {
	mock.Mock
}

// AddAmountToStake provides a mock function with given fields: ctx, acc, amt
func (_m *ReporterKeeper) AddAmountToStake(ctx context.Context, acc types.AccAddress, amt math.Int) error {
	ret := _m.Called(ctx, acc, amt)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, math.Int) error); ok {
		r0 = rf(ctx, acc, amt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delegation provides a mock function with given fields: ctx, delegator
func (_m *ReporterKeeper) Delegation(ctx context.Context, delegator types.AccAddress) (reportertypes.Selection, error) {
	ret := _m.Called(ctx, delegator)

	var r0 reportertypes.Selection
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress) (reportertypes.Selection, error)); ok {
		return rf(ctx, delegator)
	}
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress) reportertypes.Selection); ok {
		r0 = rf(ctx, delegator)
	} else {
		r0 = ret.Get(0).(reportertypes.Selection)
	}

	if rf, ok := ret.Get(1).(func(context.Context, types.AccAddress) error); ok {
		r1 = rf(ctx, delegator)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EscrowReporterStake provides a mock function with given fields: ctx, reporterAddr, power, height, amt, hashId
func (_m *ReporterKeeper) EscrowReporterStake(ctx context.Context, reporterAddr types.AccAddress, power int64, height int64, amt math.Int, hashId []byte) error {
	ret := _m.Called(ctx, reporterAddr, power, height, amt, hashId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, int64, int64, math.Int, []byte) error); ok {
		r0 = rf(ctx, reporterAddr, power, height, amt, hashId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FeeRefund provides a mock function with given fields: ctx, hashId, amt
func (_m *ReporterKeeper) FeeRefund(ctx context.Context, hashId []byte, amt math.Int) error {
	ret := _m.Called(ctx, hashId, amt)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, math.Int) error); ok {
		r0 = rf(ctx, hashId, amt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FeefromReporterStake provides a mock function with given fields: ctx, reporterAddr, amt, hashId
func (_m *ReporterKeeper) FeefromReporterStake(ctx context.Context, reporterAddr types.AccAddress, amt math.Int, hashId []byte) error {
	ret := _m.Called(ctx, reporterAddr, amt, hashId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, math.Int, []byte) error); ok {
		r0 = rf(ctx, reporterAddr, amt, hashId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDelegatorTokensAtBlock provides a mock function with given fields: ctx, delegator, blockNumber
func (_m *ReporterKeeper) GetDelegatorTokensAtBlock(ctx context.Context, delegator []byte, blockNumber int64) (math.Int, error) {
	ret := _m.Called(ctx, delegator, blockNumber)

	var r0 math.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, int64) (math.Int, error)); ok {
		return rf(ctx, delegator, blockNumber)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, int64) math.Int); ok {
		r0 = rf(ctx, delegator, blockNumber)
	} else {
		r0 = ret.Get(0).(math.Int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, int64) error); ok {
		r1 = rf(ctx, delegator, blockNumber)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetReporterTokensAtBlock provides a mock function with given fields: ctx, reporter, blockNumber
func (_m *ReporterKeeper) GetReporterTokensAtBlock(ctx context.Context, reporter []byte, blockNumber int64) (math.Int, error) {
	ret := _m.Called(ctx, reporter, blockNumber)

	var r0 math.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, int64) (math.Int, error)); ok {
		return rf(ctx, reporter, blockNumber)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, int64) math.Int); ok {
		r0 = rf(ctx, reporter, blockNumber)
	} else {
		r0 = ret.Get(0).(math.Int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, int64) error); ok {
		r1 = rf(ctx, reporter, blockNumber)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// JailReporter provides a mock function with given fields: ctx, reporterAddr, jailDuration
func (_m *ReporterKeeper) JailReporter(ctx context.Context, reporterAddr types.AccAddress, jailDuration int64) error {
	ret := _m.Called(ctx, reporterAddr, jailDuration)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, int64) error); ok {
		r0 = rf(ctx, reporterAddr, jailDuration)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReturnSlashedTokens provides a mock function with given fields: ctx, amt, hashId
func (_m *ReporterKeeper) ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) error {
	ret := _m.Called(ctx, amt, hashId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, math.Int, []byte) error); ok {
		r0 = rf(ctx, amt, hashId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TotalReporterPower provides a mock function with given fields: ctx
func (_m *ReporterKeeper) TotalReporterPower(ctx context.Context) (math.Int, error) {
	ret := _m.Called(ctx)

	var r0 math.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (math.Int, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) math.Int); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(math.Int)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewReporterKeeper interface {
	mock.TestingT
	Cleanup(func())
}

// NewReporterKeeper creates a new instance of ReporterKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewReporterKeeper(t mockConstructorTestingTNewReporterKeeper) *ReporterKeeper {
	mock := &ReporterKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
