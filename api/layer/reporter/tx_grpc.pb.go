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

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MsgClient interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
	// CreateReporter defines a (reporter) operation for creating a new reporter.
	CreateReporter(ctx context.Context, in *MsgCreateReporter, opts ...grpc.CallOption) (*MsgCreateReporterResponse, error)
	// SelectReporter defines a (selector) operation for choosing a reporter.
	SelectReporter(ctx context.Context, in *MsgSelectReporter, opts ...grpc.CallOption) (*MsgSelectReporterResponse, error)
	// SwitchReporter defines a (selector) operation for switching a reporter.
	SwitchReporter(ctx context.Context, in *MsgSwitchReporter, opts ...grpc.CallOption) (*MsgSwitchReporterResponse, error)
	// RemoveSelector defines an operation for removing a selector that no longer meets
	// the reporter's minimum requirements and the reporter is capped.
	RemoveSelector(ctx context.Context, in *MsgRemoveSelector, opts ...grpc.CallOption) (*MsgRemoveSelectorResponse, error)
	// UnjailReporter defines a method to unjail a jailed reporter.
	UnjailReporter(ctx context.Context, in *MsgUnjailReporter, opts ...grpc.CallOption) (*MsgUnjailReporterResponse, error)
	// WithdrawTip defines a method to withdraw tip from a reporter module.
	WithdrawTip(ctx context.Context, in *MsgWithdrawTip, opts ...grpc.CallOption) (*MsgWithdrawTipResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error) {
	out := new(MsgUpdateParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/UpdateParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) CreateReporter(ctx context.Context, in *MsgCreateReporter, opts ...grpc.CallOption) (*MsgCreateReporterResponse, error) {
	out := new(MsgCreateReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/CreateReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) SelectReporter(ctx context.Context, in *MsgSelectReporter, opts ...grpc.CallOption) (*MsgSelectReporterResponse, error) {
	out := new(MsgSelectReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/SelectReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) SwitchReporter(ctx context.Context, in *MsgSwitchReporter, opts ...grpc.CallOption) (*MsgSwitchReporterResponse, error) {
	out := new(MsgSwitchReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/SwitchReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) RemoveSelector(ctx context.Context, in *MsgRemoveSelector, opts ...grpc.CallOption) (*MsgRemoveSelectorResponse, error) {
	out := new(MsgRemoveSelectorResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/RemoveSelector", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UnjailReporter(ctx context.Context, in *MsgUnjailReporter, opts ...grpc.CallOption) (*MsgUnjailReporterResponse, error) {
	out := new(MsgUnjailReporterResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/UnjailReporter", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) WithdrawTip(ctx context.Context, in *MsgWithdrawTip, opts ...grpc.CallOption) (*MsgWithdrawTipResponse, error) {
	out := new(MsgWithdrawTipResponse)
	err := c.cc.Invoke(ctx, "/layer.reporter.Msg/WithdrawTip", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility
type MsgServer interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	// CreateReporter defines a (reporter) operation for creating a new reporter.
	CreateReporter(context.Context, *MsgCreateReporter) (*MsgCreateReporterResponse, error)
	// SelectReporter defines a (selector) operation for choosing a reporter.
	SelectReporter(context.Context, *MsgSelectReporter) (*MsgSelectReporterResponse, error)
	// SwitchReporter defines a (selector) operation for switching a reporter.
	SwitchReporter(context.Context, *MsgSwitchReporter) (*MsgSwitchReporterResponse, error)
	// RemoveSelector defines an operation for removing a selector that no longer meets
	// the reporter's minimum requirements and the reporter is capped.
	RemoveSelector(context.Context, *MsgRemoveSelector) (*MsgRemoveSelectorResponse, error)
	// UnjailReporter defines a method to unjail a jailed reporter.
	UnjailReporter(context.Context, *MsgUnjailReporter) (*MsgUnjailReporterResponse, error)
	// WithdrawTip defines a method to withdraw tip from a reporter module.
	WithdrawTip(context.Context, *MsgWithdrawTip) (*MsgWithdrawTipResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (UnimplementedMsgServer) CreateReporter(context.Context, *MsgCreateReporter) (*MsgCreateReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateReporter not implemented")
}
func (UnimplementedMsgServer) SelectReporter(context.Context, *MsgSelectReporter) (*MsgSelectReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SelectReporter not implemented")
}
func (UnimplementedMsgServer) SwitchReporter(context.Context, *MsgSwitchReporter) (*MsgSwitchReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SwitchReporter not implemented")
}
func (UnimplementedMsgServer) RemoveSelector(context.Context, *MsgRemoveSelector) (*MsgRemoveSelectorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveSelector not implemented")
}
func (UnimplementedMsgServer) UnjailReporter(context.Context, *MsgUnjailReporter) (*MsgUnjailReporterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnjailReporter not implemented")
}
func (UnimplementedMsgServer) WithdrawTip(context.Context, *MsgWithdrawTip) (*MsgWithdrawTipResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WithdrawTip not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/UpdateParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_CreateReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateReporter)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/CreateReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateReporter(ctx, req.(*MsgCreateReporter))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SelectReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSelectReporter)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SelectReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/SelectReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SelectReporter(ctx, req.(*MsgSelectReporter))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SwitchReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSwitchReporter)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SwitchReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/SwitchReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SwitchReporter(ctx, req.(*MsgSwitchReporter))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_RemoveSelector_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRemoveSelector)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RemoveSelector(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/RemoveSelector",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RemoveSelector(ctx, req.(*MsgRemoveSelector))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UnjailReporter_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUnjailReporter)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UnjailReporter(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/UnjailReporter",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UnjailReporter(ctx, req.(*MsgUnjailReporter))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_WithdrawTip_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgWithdrawTip)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).WithdrawTip(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.reporter.Msg/WithdrawTip",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).WithdrawTip(ctx, req.(*MsgWithdrawTip))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.reporter.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
		{
			MethodName: "CreateReporter",
			Handler:    _Msg_CreateReporter_Handler,
		},
		{
			MethodName: "SelectReporter",
			Handler:    _Msg_SelectReporter_Handler,
		},
		{
			MethodName: "SwitchReporter",
			Handler:    _Msg_SwitchReporter_Handler,
		},
		{
			MethodName: "RemoveSelector",
			Handler:    _Msg_RemoveSelector_Handler,
		},
		{
			MethodName: "UnjailReporter",
			Handler:    _Msg_UnjailReporter_Handler,
		},
		{
			MethodName: "WithdrawTip",
			Handler:    _Msg_WithdrawTip_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/reporter/tx.proto",
}
