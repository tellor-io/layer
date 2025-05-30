// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	types "github.com/tellor-io/layer/daemons/server/types/daemons"
)

// QueryClient is an autogenerated mock type for the QueryClient type
type QueryClient struct {
	mock.Mock
}

// UpdateMarketPrices provides a mock function with given fields: ctx, in, opts
func (_m *QueryClient) UpdateMarketPrices(ctx context.Context, in *types.UpdateMarketPricesRequest, opts ...grpc.CallOption) (*types.UpdateMarketPricesResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for UpdateMarketPrices")
	}

	var r0 *types.UpdateMarketPricesResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.UpdateMarketPricesRequest, ...grpc.CallOption) (*types.UpdateMarketPricesResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.UpdateMarketPricesRequest, ...grpc.CallOption) *types.UpdateMarketPricesResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.UpdateMarketPricesResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.UpdateMarketPricesRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewQueryClient creates a new instance of QueryClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewQueryClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *QueryClient {
	mock := &QueryClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
