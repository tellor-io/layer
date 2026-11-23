package mocks

import (
	"bytes"
	"context"

	"github.com/stretchr/testify/mock"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ReporterKeeper preserves historical vote defaults across Mockery generation.
// Tests can override the defaults with explicit expectations.
type ReporterKeeper struct {
	reporterKeeperMock
}

// expectationSet reports whether a test registered an On for method.
// History lookups default to "live selection is accurate" and "no switch was
// unfinished" so existing vote tests keep the pre-history behavior without a
// new expectation.
func (_m *ReporterKeeper) expectationSet(method string) bool {
	for _, call := range _m.ExpectedCalls {
		if call.Method == method {
			return true
		}
	}
	return false
}

// delegatedReporter returns the reporter from a Delegation expectation for selector.
func (_m *ReporterKeeper) delegatedReporter(selector sdk.AccAddress) (sdk.AccAddress, bool) {
	for i := len(_m.ExpectedCalls) - 1; i >= 0; i-- {
		call := _m.ExpectedCalls[i]
		if call.Method != "Delegation" || len(call.Arguments) < 2 {
			continue
		}
		arg, ok := call.Arguments.Get(1).(sdk.AccAddress)
		if !ok || !bytes.Equal(arg, selector) {
			continue
		}
		if len(call.ReturnArguments) == 0 || call.ReturnArguments.Get(0) == nil {
			continue
		}
		sel, ok := call.ReturnArguments.Get(0).(reportertypes.Selection)
		if !ok || len(sel.Reporter) == 0 {
			continue
		}
		return append(sdk.AccAddress(nil), sel.Reporter...), true
	}
	return nil, false
}

// ReporterAtHeight provides a mock function with given fields: ctx, selector, blockHeight.
// With no expectation, the selector's Delegation reporter is the dispute-height
// selection and live data is accurate. An address with no Delegation expectation
// is treated as selected to itself, so a reporter owner still votes the group.
func (_m *ReporterKeeper) ReporterAtHeight(ctx context.Context, selector sdk.AccAddress, blockHeight uint64) (sdk.AccAddress, bool, error) {
	if !_m.expectationSet("ReporterAtHeight") {
		if rep, ok := _m.delegatedReporter(selector); ok {
			return rep, true, nil
		}
		return append(sdk.AccAddress(nil), selector...), true, nil
	}
	ret := _m.Called(ctx, selector, blockHeight)

	if rf, ok := ret.Get(0).(func(context.Context, sdk.AccAddress, uint64) (sdk.AccAddress, bool, error)); ok {
		return rf(ctx, selector, blockHeight)
	}
	var r0 sdk.AccAddress
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(sdk.AccAddress)
	}
	r1, _ := ret.Get(1).(bool)
	return r0, r1, ret.Error(2)
}

// PruneSelectorSwitches provides a mock function with given fields: ctx, pruneHeight, limit.
// With no expectation, pruning is a no-op so end-blocker tests do not have to expect it.
func (_m *ReporterKeeper) PruneSelectorSwitches(ctx context.Context, pruneHeight uint64, limit int) error {
	if !_m.expectationSet("PruneSelectorSwitches") {
		return nil
	}
	ret := _m.Called(ctx, pruneHeight, limit)
	if rf, ok := ret.Get(0).(func(context.Context, uint64, int) error); ok {
		return rf(ctx, pruneHeight, limit)
	}
	return ret.Error(0)
}

// SelectorSwitchUnfinishedAt provides a mock function with given fields: ctx, selector, blockHeight.
// With no expectation, no switch was unfinished at blockHeight.
func (_m *ReporterKeeper) SelectorSwitchUnfinishedAt(ctx context.Context, selector sdk.AccAddress, blockHeight uint64) (bool, error) {
	if !_m.expectationSet("SelectorSwitchUnfinishedAt") {
		return false, nil
	}
	ret := _m.Called(ctx, selector, blockHeight)

	if rf, ok := ret.Get(0).(func(context.Context, sdk.AccAddress, uint64) (bool, error)); ok {
		return rf(ctx, selector, blockHeight)
	}
	r0, _ := ret.Get(0).(bool)
	return r0, ret.Error(1)
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
