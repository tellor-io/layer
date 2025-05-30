// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package registry

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
	// Queries a list of GetDataSpec items.
	GetDataSpec(ctx context.Context, in *QueryGetDataSpecRequest, opts ...grpc.CallOption) (*QueryGetDataSpecResponse, error)
	// Queries a list of DecodeQuerydata items.
	DecodeQuerydata(ctx context.Context, in *QueryDecodeQuerydataRequest, opts ...grpc.CallOption) (*QueryDecodeQuerydataResponse, error)
	// Queries a list of GenerateQuerydata items.
	GenerateQuerydata(ctx context.Context, in *QueryGenerateQuerydataRequest, opts ...grpc.CallOption) (*QueryGenerateQuerydataResponse, error)
	// Queries a list of DecodeValue items.
	DecodeValue(ctx context.Context, in *QueryDecodeValueRequest, opts ...grpc.CallOption) (*QueryDecodeValueResponse, error)
	// Queries a list of GetAllDataSpecs items.
	GetAllDataSpecs(ctx context.Context, in *QueryGetAllDataSpecsRequest, opts ...grpc.CallOption) (*QueryGetAllDataSpecsResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetDataSpec(ctx context.Context, in *QueryGetDataSpecRequest, opts ...grpc.CallOption) (*QueryGetDataSpecResponse, error) {
	out := new(QueryGetDataSpecResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Query/GetDataSpec", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) DecodeQuerydata(ctx context.Context, in *QueryDecodeQuerydataRequest, opts ...grpc.CallOption) (*QueryDecodeQuerydataResponse, error) {
	out := new(QueryDecodeQuerydataResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Query/DecodeQuerydata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GenerateQuerydata(ctx context.Context, in *QueryGenerateQuerydataRequest, opts ...grpc.CallOption) (*QueryGenerateQuerydataResponse, error) {
	out := new(QueryGenerateQuerydataResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Query/GenerateQuerydata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) DecodeValue(ctx context.Context, in *QueryDecodeValueRequest, opts ...grpc.CallOption) (*QueryDecodeValueResponse, error) {
	out := new(QueryDecodeValueResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Query/DecodeValue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetAllDataSpecs(ctx context.Context, in *QueryGetAllDataSpecsRequest, opts ...grpc.CallOption) (*QueryGetAllDataSpecsResponse, error) {
	out := new(QueryGetAllDataSpecsResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Query/GetAllDataSpecs", in, out, opts...)
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
	// Queries a list of GetDataSpec items.
	GetDataSpec(context.Context, *QueryGetDataSpecRequest) (*QueryGetDataSpecResponse, error)
	// Queries a list of DecodeQuerydata items.
	DecodeQuerydata(context.Context, *QueryDecodeQuerydataRequest) (*QueryDecodeQuerydataResponse, error)
	// Queries a list of GenerateQuerydata items.
	GenerateQuerydata(context.Context, *QueryGenerateQuerydataRequest) (*QueryGenerateQuerydataResponse, error)
	// Queries a list of DecodeValue items.
	DecodeValue(context.Context, *QueryDecodeValueRequest) (*QueryDecodeValueResponse, error)
	// Queries a list of GetAllDataSpecs items.
	GetAllDataSpecs(context.Context, *QueryGetAllDataSpecsRequest) (*QueryGetAllDataSpecsResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) GetDataSpec(context.Context, *QueryGetDataSpecRequest) (*QueryGetDataSpecResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDataSpec not implemented")
}
func (UnimplementedQueryServer) DecodeQuerydata(context.Context, *QueryDecodeQuerydataRequest) (*QueryDecodeQuerydataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecodeQuerydata not implemented")
}
func (UnimplementedQueryServer) GenerateQuerydata(context.Context, *QueryGenerateQuerydataRequest) (*QueryGenerateQuerydataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateQuerydata not implemented")
}
func (UnimplementedQueryServer) DecodeValue(context.Context, *QueryDecodeValueRequest) (*QueryDecodeValueResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DecodeValue not implemented")
}
func (UnimplementedQueryServer) GetAllDataSpecs(context.Context, *QueryGetAllDataSpecsRequest) (*QueryGetAllDataSpecsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAllDataSpecs not implemented")
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
		FullMethod: "/layer.registry.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetDataSpec_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetDataSpecRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetDataSpec(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Query/GetDataSpec",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetDataSpec(ctx, req.(*QueryGetDataSpecRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_DecodeQuerydata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDecodeQuerydataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DecodeQuerydata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Query/DecodeQuerydata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DecodeQuerydata(ctx, req.(*QueryDecodeQuerydataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GenerateQuerydata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGenerateQuerydataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GenerateQuerydata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Query/GenerateQuerydata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GenerateQuerydata(ctx, req.(*QueryGenerateQuerydataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_DecodeValue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDecodeValueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DecodeValue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Query/DecodeValue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DecodeValue(ctx, req.(*QueryDecodeValueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetAllDataSpecs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetAllDataSpecsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetAllDataSpecs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Query/GetAllDataSpecs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetAllDataSpecs(ctx, req.(*QueryGetAllDataSpecsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.registry.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "GetDataSpec",
			Handler:    _Query_GetDataSpec_Handler,
		},
		{
			MethodName: "DecodeQuerydata",
			Handler:    _Query_DecodeQuerydata_Handler,
		},
		{
			MethodName: "GenerateQuerydata",
			Handler:    _Query_GenerateQuerydata_Handler,
		},
		{
			MethodName: "DecodeValue",
			Handler:    _Query_DecodeValue_Handler,
		},
		{
			MethodName: "GetAllDataSpecs",
			Handler:    _Query_GetAllDataSpecs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/registry/query.proto",
}
