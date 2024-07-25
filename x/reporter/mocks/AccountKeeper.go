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

// GetAccount provides a mock function with given fields: _a0, _a1
func (_m *AccountKeeper) GetAccount(_a0 context.Context, _a1 types.AccAddress) types.AccountI {
	ret := _m.Called(_a0, _a1)

	var r0 types.AccountI
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress) types.AccountI); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.AccountI)
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