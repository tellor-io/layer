// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	cosmos_sdktypes "github.com/cosmos/cosmos-sdk/types"

	math "cosmossdk.io/math"

	mock "github.com/stretchr/testify/mock"

	types "github.com/tellor-io/layer/x/oracle/types"
)

// OracleKeeper is an autogenerated mock type for the OracleKeeper type
type OracleKeeper struct {
	mock.Mock
}

// FlagAggregateReport provides a mock function with given fields: ctx, report
func (_m *OracleKeeper) FlagAggregateReport(ctx context.Context, report types.MicroReport) error {
	ret := _m.Called(ctx, report)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.MicroReport) error); ok {
		r0 = rf(ctx, report)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetTipsAtBlockForTipper provides a mock function with given fields: ctx, blockNumber, tipper
func (_m *OracleKeeper) GetTipsAtBlockForTipper(ctx context.Context, blockNumber uint64, tipper cosmos_sdktypes.AccAddress) (math.Int, error) {
	ret := _m.Called(ctx, blockNumber, tipper)

	var r0 math.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, cosmos_sdktypes.AccAddress) (math.Int, error)); ok {
		return rf(ctx, blockNumber, tipper)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64, cosmos_sdktypes.AccAddress) math.Int); ok {
		r0 = rf(ctx, blockNumber, tipper)
	} else {
		r0 = ret.Get(0).(math.Int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64, cosmos_sdktypes.AccAddress) error); ok {
		r1 = rf(ctx, blockNumber, tipper)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTotalTips provides a mock function with given fields: ctx
func (_m *OracleKeeper) GetTotalTips(ctx context.Context) (math.Int, error) {
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

// GetTotalTipsAtBlock provides a mock function with given fields: ctx, blockNumber
func (_m *OracleKeeper) GetTotalTipsAtBlock(ctx context.Context, blockNumber uint64) (math.Int, error) {
	ret := _m.Called(ctx, blockNumber)

	var r0 math.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (math.Int, error)); ok {
		return rf(ctx, blockNumber)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) math.Int); ok {
		r0 = rf(ctx, blockNumber)
	} else {
		r0 = ret.Get(0).(math.Int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, blockNumber)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserTips provides a mock function with given fields: ctx, tipper
func (_m *OracleKeeper) GetUserTips(ctx context.Context, tipper cosmos_sdktypes.AccAddress) (math.Int, error) {
	ret := _m.Called(ctx, tipper)

	var r0 math.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, cosmos_sdktypes.AccAddress) (math.Int, error)); ok {
		return rf(ctx, tipper)
	}
	if rf, ok := ret.Get(0).(func(context.Context, cosmos_sdktypes.AccAddress) math.Int); ok {
		r0 = rf(ctx, tipper)
	} else {
		r0 = ret.Get(0).(math.Int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, cosmos_sdktypes.AccAddress) error); ok {
		r1 = rf(ctx, tipper)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewOracleKeeper interface {
	mock.TestingT
	Cleanup(func())
}

// NewOracleKeeper creates a new instance of OracleKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewOracleKeeper(t mockConstructorTestingTNewOracleKeeper) *OracleKeeper {
	mock := &OracleKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
