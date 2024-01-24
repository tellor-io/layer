// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"
)

// GrpcClient is an autogenerated mock type for the GrpcClient type
type GrpcClient struct {
	mock.Mock
}

// CloseConnection provides a mock function with given fields: grpcConn
func (_m *GrpcClient) CloseConnection(grpcConn *grpc.ClientConn) error {
	ret := _m.Called(grpcConn)

	var r0 error
	if rf, ok := ret.Get(0).(func(*grpc.ClientConn) error); ok {
		r0 = rf(grpcConn)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewGrpcConnection provides a mock function with given fields: ctx, socketAddress
func (_m *GrpcClient) NewGrpcConnection(ctx context.Context, socketAddress string) (*grpc.ClientConn, error) {
	ret := _m.Called(ctx, socketAddress)

	var r0 *grpc.ClientConn
	if rf, ok := ret.Get(0).(func(context.Context, string) *grpc.ClientConn); ok {
		r0 = rf(ctx, socketAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*grpc.ClientConn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, socketAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewTcpConnection provides a mock function with given fields: ctx, endpoint
func (_m *GrpcClient) NewTcpConnection(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	ret := _m.Called(ctx, endpoint)

	var r0 *grpc.ClientConn
	if rf, ok := ret.Get(0).(func(context.Context, string) *grpc.ClientConn); ok {
		r0 = rf(ctx, endpoint)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*grpc.ClientConn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, endpoint)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewGrpcClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewGrpcClient creates a new instance of GrpcClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGrpcClient(t mockConstructorTestingTNewGrpcClient) *GrpcClient {
	mock := &GrpcClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
