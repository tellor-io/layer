// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package bridge

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
	// Queries a list of GetEvmValidators items.
	GetEvmValidators(ctx context.Context, in *QueryGetEvmValidatorsRequest, opts ...grpc.CallOption) (*QueryGetEvmValidatorsResponse, error)
	GetValidatorCheckpoint(ctx context.Context, in *QueryGetValidatorCheckpointRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointResponse, error)
	GetValidatorCheckpointParams(ctx context.Context, in *QueryGetValidatorCheckpointParamsRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointParamsResponse, error)
	GetValidatorTimestampByIndex(ctx context.Context, in *QueryGetValidatorTimestampByIndexRequest, opts ...grpc.CallOption) (*QueryGetValidatorTimestampByIndexResponse, error)
	GetValsetSigs(ctx context.Context, in *QueryGetValsetSigsRequest, opts ...grpc.CallOption) (*QueryGetValsetSigsResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/Params", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetEvmValidators(ctx context.Context, in *QueryGetEvmValidatorsRequest, opts ...grpc.CallOption) (*QueryGetEvmValidatorsResponse, error) {
	out := new(QueryGetEvmValidatorsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetEvmValidators", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorCheckpoint(ctx context.Context, in *QueryGetValidatorCheckpointRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointResponse, error) {
	out := new(QueryGetValidatorCheckpointResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorCheckpoint", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorCheckpointParams(ctx context.Context, in *QueryGetValidatorCheckpointParamsRequest, opts ...grpc.CallOption) (*QueryGetValidatorCheckpointParamsResponse, error) {
	out := new(QueryGetValidatorCheckpointParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorCheckpointParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValidatorTimestampByIndex(ctx context.Context, in *QueryGetValidatorTimestampByIndexRequest, opts ...grpc.CallOption) (*QueryGetValidatorTimestampByIndexResponse, error) {
	out := new(QueryGetValidatorTimestampByIndexResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValidatorTimestampByIndex", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) GetValsetSigs(ctx context.Context, in *QueryGetValsetSigsRequest, opts ...grpc.CallOption) (*QueryGetValsetSigsResponse, error) {
	out := new(QueryGetValsetSigsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Query/GetValsetSigs", in, out, opts...)
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
	// Queries a list of GetEvmValidators items.
	GetEvmValidators(context.Context, *QueryGetEvmValidatorsRequest) (*QueryGetEvmValidatorsResponse, error)
	GetValidatorCheckpoint(context.Context, *QueryGetValidatorCheckpointRequest) (*QueryGetValidatorCheckpointResponse, error)
	GetValidatorCheckpointParams(context.Context, *QueryGetValidatorCheckpointParamsRequest) (*QueryGetValidatorCheckpointParamsResponse, error)
	GetValidatorTimestampByIndex(context.Context, *QueryGetValidatorTimestampByIndexRequest) (*QueryGetValidatorTimestampByIndexResponse, error)
	GetValsetSigs(context.Context, *QueryGetValsetSigsRequest) (*QueryGetValsetSigsResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) GetEvmValidators(context.Context, *QueryGetEvmValidatorsRequest) (*QueryGetEvmValidatorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEvmValidators not implemented")
}
func (UnimplementedQueryServer) GetValidatorCheckpoint(context.Context, *QueryGetValidatorCheckpointRequest) (*QueryGetValidatorCheckpointResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorCheckpoint not implemented")
}
func (UnimplementedQueryServer) GetValidatorCheckpointParams(context.Context, *QueryGetValidatorCheckpointParamsRequest) (*QueryGetValidatorCheckpointParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorCheckpointParams not implemented")
}
func (UnimplementedQueryServer) GetValidatorTimestampByIndex(context.Context, *QueryGetValidatorTimestampByIndexRequest) (*QueryGetValidatorTimestampByIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValidatorTimestampByIndex not implemented")
}
func (UnimplementedQueryServer) GetValsetSigs(context.Context, *QueryGetValsetSigsRequest) (*QueryGetValsetSigsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetValsetSigs not implemented")
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
		FullMethod: "/layer.bridge.Query/Params",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetEvmValidators_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetEvmValidatorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetEvmValidators(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetEvmValidators",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetEvmValidators(ctx, req.(*QueryGetEvmValidatorsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorCheckpoint_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorCheckpointRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorCheckpoint(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorCheckpoint",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorCheckpoint(ctx, req.(*QueryGetValidatorCheckpointRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorCheckpointParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorCheckpointParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorCheckpointParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorCheckpointParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorCheckpointParams(ctx, req.(*QueryGetValidatorCheckpointParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValidatorTimestampByIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValidatorTimestampByIndexRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValidatorTimestampByIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValidatorTimestampByIndex",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValidatorTimestampByIndex(ctx, req.(*QueryGetValidatorTimestampByIndexRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_GetValsetSigs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryGetValsetSigsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).GetValsetSigs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Query/GetValsetSigs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).GetValsetSigs(ctx, req.(*QueryGetValsetSigsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.bridge.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "GetEvmValidators",
			Handler:    _Query_GetEvmValidators_Handler,
		},
		{
			MethodName: "GetValidatorCheckpoint",
			Handler:    _Query_GetValidatorCheckpoint_Handler,
		},
		{
			MethodName: "GetValidatorCheckpointParams",
			Handler:    _Query_GetValidatorCheckpointParams_Handler,
		},
		{
			MethodName: "GetValidatorTimestampByIndex",
			Handler:    _Query_GetValidatorTimestampByIndex_Handler,
		},
		{
			MethodName: "GetValsetSigs",
			Handler:    _Query_GetValsetSigs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/bridge/query.proto",
}
