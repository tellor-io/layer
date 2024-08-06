// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package reporter

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// Parameters queries the parameters of the module.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// Reporters queries all the staked reporters.
	Reporters(ctx context.Context, in *QueryReportersRequest, opts ...grpc.CallOption) (*QueryReportersResponse, error)
	// SelectorReporter queries the reporter of a selector.
	SelectorReporter(ctx context.Context, in *QuerySelectorReporterRequest, opts ...grpc.CallOption) (*QuerySelectorReporterResponse, error)
	// AllowedAmount queries the currently allowed amount to stake or unstake.
	AllowedAmount(ctx context.Context, in *QueryAllowedAmountRequest, opts ...grpc.CallOption) (*QueryAllowedAmountResponse, error)
	AllowedAmountExpiration(ctx context.Context, in *QueryAllowedAmountExpirationRequest, opts ...grpc.CallOption) (*QueryAllowedAmountExpirationResponse, error)
	// NumOfSelectorsByReporter queries the number of selectors by a reporter.
	NumOfSelectorsByReporter(ctx context.Context, in *QueryNumOfSelectorsByReporterRequest, opts ...grpc.CallOption) (*QueryNumOfSelectorsByReporterResponse, error)
	// SpaceAvailableByReporter queries the space available in a reporter.
	SpaceAvailableByReporter(ctx context.Context, in *QuerySpaceAvailableByReporterRequest, opts ...grpc.CallOption) (*QuerySpaceAvailableByReporterResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Reporters(ctx context.Context, in *QueryReportersRequest, opts ...grpc.CallOption) (*QueryReportersResponse, error) {
	out := new(QueryReportersResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/Reporters", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) SelectorReporter(ctx context.Context, in *QuerySelectorReporterRequest, opts ...grpc.CallOption) (*QuerySelectorReporterResponse, error) {
	out := new(QuerySelectorReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/SelectorReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AllowedAmount(ctx context.Context, in *QueryAllowedAmountRequest, opts ...grpc.CallOption) (*QueryAllowedAmountResponse, error) {
	out := new(QueryAllowedAmountResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/AllowedAmount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) AllowedAmountExpiration(ctx context.Context, in *QueryAllowedAmountExpirationRequest, opts ...grpc.CallOption) (*QueryAllowedAmountExpirationResponse, error) {
	out := new(QueryAllowedAmountExpirationResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/AllowedAmountExpiration", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) NumOfSelectorsByReporter(ctx context.Context, in *QueryNumOfSelectorsByReporterRequest, opts ...grpc.CallOption) (*QueryNumOfSelectorsByReporterResponse, error) {
	out := new(QueryNumOfSelectorsByReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/NumOfSelectorsByReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) SpaceAvailableByReporter(ctx context.Context, in *QuerySpaceAvailableByReporterRequest, opts ...grpc.CallOption) (*QuerySpaceAvailableByReporterResponse, error) {
	out := new(QuerySpaceAvailableByReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Query/SpaceAvailableByReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// Parameters queries the parameters of the module.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	// Reporters queries all the staked reporters.
	Reporters(context.Context, *QueryReportersRequest) (*QueryReportersResponse, error)
	// SelectorReporter queries the reporter of a selector.
	SelectorReporter(context.Context, *QuerySelectorReporterRequest) (*QuerySelectorReporterResponse, error)
	// AllowedAmount queries the currently allowed amount to stake or unstake.
	AllowedAmount(context.Context, *QueryAllowedAmountRequest) (*QueryAllowedAmountResponse, error)
	AllowedAmountExpiration(context.Context, *QueryAllowedAmountExpirationRequest) (*QueryAllowedAmountExpirationResponse, error)
	// NumOfSelectorsByReporter queries the number of selectors by a reporter.
	NumOfSelectorsByReporter(context.Context, *QueryNumOfSelectorsByReporterRequest) (*QueryNumOfSelectorsByReporterResponse, error)
	// SpaceAvailableByReporter queries the space available in a reporter.
	SpaceAvailableByReporter(context.Context, *QuerySpaceAvailableByReporterRequest) (*QuerySpaceAvailableByReporterResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) Reporters(context.Context, *QueryReportersRequest) (*QueryReportersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Reporters not implemented")
}
func (UnimplementedQueryServer) SelectorReporter(context.Context, *QuerySelectorReporterRequest) (*QuerySelectorReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SelectorReporter not implemented")
}
func (UnimplementedQueryServer) AllowedAmount(context.Context, *QueryAllowedAmountRequest) (*QueryAllowedAmountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AllowedAmount not implemented")
}
func (UnimplementedQueryServer) AllowedAmountExpiration(context.Context, *QueryAllowedAmountExpirationRequest) (*QueryAllowedAmountExpirationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AllowedAmountExpiration not implemented")
}
func (UnimplementedQueryServer) NumOfSelectorsByReporter(context.Context, *QueryNumOfSelectorsByReporterRequest) (*QueryNumOfSelectorsByReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NumOfSelectorsByReporter not implemented")
}
func (UnimplementedQueryServer) SpaceAvailableByReporter(context.Context, *QuerySpaceAvailableByReporterRequest) (*QuerySpaceAvailableByReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SpaceAvailableByReporter not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Reporters_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryReportersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Reporters(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/Reporters",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Reporters(ctx, req.(*QueryReportersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_SelectorReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySelectorReporterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).SelectorReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/SelectorReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).SelectorReporter(ctx, req.(*QuerySelectorReporterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AllowedAmount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllowedAmountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AllowedAmount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/AllowedAmount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AllowedAmount(ctx, req.(*QueryAllowedAmountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_AllowedAmountExpiration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllowedAmountExpirationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AllowedAmountExpiration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/AllowedAmountExpiration",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AllowedAmountExpiration(ctx, req.(*QueryAllowedAmountExpirationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_NumOfSelectorsByReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryNumOfSelectorsByReporterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).NumOfSelectorsByReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/NumOfSelectorsByReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).NumOfSelectorsByReporter(ctx, req.(*QueryNumOfSelectorsByReporterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_SpaceAvailableByReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySpaceAvailableByReporterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).SpaceAvailableByReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Query/SpaceAvailableByReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).SpaceAvailableByReporter(ctx, req.(*QuerySpaceAvailableByReporterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.reporter.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "Reporters",
			Handler:    _Query_Reporters_Handler,
		},
		{
			MethodName: "SelectorReporter",
			Handler:    _Query_SelectorReporter_Handler,
		},
		{
			MethodName: "AllowedAmount",
			Handler:    _Query_AllowedAmount_Handler,
		},
		{
			MethodName: "AllowedAmountExpiration",
			Handler:    _Query_AllowedAmountExpiration_Handler,
		},
		{
			MethodName: "NumOfSelectorsByReporter",
			Handler:    _Query_NumOfSelectorsByReporter_Handler,
		},
		{
			MethodName: "SpaceAvailableByReporter",
			Handler:    _Query_SpaceAvailableByReporter_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/reporter/query.proto",
}
