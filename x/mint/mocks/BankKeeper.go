// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	cosmos_sdktypes "github.com/cosmos/cosmos-sdk/types"

	mock "github.com/stretchr/testify/mock"

	types "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// BankKeeper is an autogenerated mock type for the BankKeeper type
type BankKeeper struct {
	mock.Mock
}

// InputOutputCoins provides a mock function with given fields: ctx, inputs, outputs
func (_m *BankKeeper) InputOutputCoins(ctx context.Context, inputs types.Input, outputs []types.Output) error {
	ret := _m.Called(ctx, inputs, outputs)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.Input, []types.Output) error); ok {
		r0 = rf(ctx, inputs, outputs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MintCoins provides a mock function with given fields: ctx, name, amt
func (_m *BankKeeper) MintCoins(ctx context.Context, name string, amt cosmos_sdktypes.Coins) error {
	ret := _m.Called(ctx, name, amt)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, cosmos_sdktypes.Coins) error); ok {
		r0 = rf(ctx, name, amt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendCoinsFromModuleToModule provides a mock function with given fields: ctx, senderModule, recipientModule, amt
func (_m *BankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt cosmos_sdktypes.Coins) error {
	ret := _m.Called(ctx, senderModule, recipientModule, amt)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, cosmos_sdktypes.Coins) error); ok {
		r0 = rf(ctx, senderModule, recipientModule, amt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewBankKeeper interface {
	mock.TestingT
	Cleanup(func())
}

// NewBankKeeper creates a new instance of BankKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBankKeeper(t mockConstructorTestingTNewBankKeeper) *BankKeeper {
	mock := &BankKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}