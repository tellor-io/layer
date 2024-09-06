// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	types "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper is an autogenerated mock type for the AccountKeeper type
type AccountKeeper struct {
	mock.Mock
}

// GetModuleAccount provides a mock function with given fields: ctx, moduleName
func (_m *AccountKeeper) GetModuleAccount(ctx context.Context, moduleName string) types.ModuleAccountI {
	ret := _m.Called(ctx, moduleName)

	var r0 types.ModuleAccountI
	if rf, ok := ret.Get(0).(func(context.Context, string) types.ModuleAccountI); ok {
		r0 = rf(ctx, moduleName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ModuleAccountI)
		}
	}

	return r0
}

// GetModuleAddress provides a mock function with given fields: name
func (_m *AccountKeeper) GetModuleAddress(name string) types.AccAddress {
	ret := _m.Called(name)

	var r0 types.AccAddress
	if rf, ok := ret.Get(0).(func(string) types.AccAddress); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.AccAddress)
		}
	}

	return r0
}

type mockConstructorTestingTNewAccountKeeper interface {
	mock.TestingT
	Cleanup(func())
}

// NewAccountKeeper creates a new instance of AccountKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAccountKeeper(t mockConstructorTestingTNewAccountKeeper) *AccountKeeper {
	mock := &AccountKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
