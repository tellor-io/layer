// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
	pricefeedtypes "github.com/tellor-io/layer/daemons/pricefeed/types"

	types "github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

// ExchangeToMarketPrices is an autogenerated mock type for the ExchangeToMarketPrices type
type ExchangeToMarketPrices struct {
	mock.Mock
}

// GetAllPrices provides a mock function with no fields
func (_m *ExchangeToMarketPrices) GetAllPrices() map[string][]types.MarketPriceTimestamp {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAllPrices")
	}

	var r0 map[string][]types.MarketPriceTimestamp
	if rf, ok := ret.Get(0).(func() map[string][]types.MarketPriceTimestamp); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]types.MarketPriceTimestamp)
		}
	}

	return r0
}

// GetIndexPrice provides a mock function with given fields: marketId, cutoffTime, resolver
func (_m *ExchangeToMarketPrices) GetIndexPrice(marketId uint32, cutoffTime time.Time, resolver pricefeedtypes.Resolver) (uint64, int) {
	ret := _m.Called(marketId, cutoffTime, resolver)

	if len(ret) == 0 {
		panic("no return value specified for GetIndexPrice")
	}

	var r0 uint64
	var r1 int
	if rf, ok := ret.Get(0).(func(uint32, time.Time, pricefeedtypes.Resolver) (uint64, int)); ok {
		return rf(marketId, cutoffTime, resolver)
	}
	if rf, ok := ret.Get(0).(func(uint32, time.Time, pricefeedtypes.Resolver) uint64); ok {
		r0 = rf(marketId, cutoffTime, resolver)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(uint32, time.Time, pricefeedtypes.Resolver) int); ok {
		r1 = rf(marketId, cutoffTime, resolver)
	} else {
		r1 = ret.Get(1).(int)
	}

	return r0, r1
}

// UpdatePrice provides a mock function with given fields: exchangeId, marketPriceTimestamp
func (_m *ExchangeToMarketPrices) UpdatePrice(exchangeId string, marketPriceTimestamp *types.MarketPriceTimestamp) {
	_m.Called(exchangeId, marketPriceTimestamp)
}

// NewExchangeToMarketPrices creates a new instance of ExchangeToMarketPrices. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewExchangeToMarketPrices(t interface {
	mock.TestingT
	Cleanup(func())
}) *ExchangeToMarketPrices {
	mock := &ExchangeToMarketPrices{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
