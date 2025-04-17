// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// BridgeKeeper is an autogenerated mock type for the BridgeKeeper type
type BridgeKeeper struct {
	mock.Mock
}

// ClaimDeposit provides a mock function with given fields: ctx, depositId, timestamp
func (_m *BridgeKeeper) ClaimDeposit(ctx context.Context, depositId uint64, timestamp uint64) error {
	ret := _m.Called(ctx, depositId, timestamp)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, uint64) error); ok {
		r0 = rf(ctx, depositId, timestamp)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDepositStatus provides a mock function with given fields: ctx, depositId
func (_m *BridgeKeeper) GetDepositStatus(ctx context.Context, depositId uint64) (bool, error) {
	ret := _m.Called(ctx, depositId)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (bool, error)); ok {
		return rf(ctx, depositId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) bool); ok {
		r0 = rf(ctx, depositId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, depositId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
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
