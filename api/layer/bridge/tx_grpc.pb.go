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

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MsgClient interface {
	// UpdateParams defines a (governance) operation for updating the module
	// parameters. The authority defaults to the x/gov module account.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
	RequestAttestations(ctx context.Context, in *MsgRequestAttestations, opts ...grpc.CallOption) (*MsgRequestAttestationsResponse, error)
	WithdrawTokens(ctx context.Context, in *MsgWithdrawTokens, opts ...grpc.CallOption) (*MsgWithdrawTokensResponse, error)
	ClaimDeposits(ctx context.Context, in *MsgClaimDepositsRequest, opts ...grpc.CallOption) (*MsgClaimDepositsResponse, error)
	UpdateSnapshotLimit(ctx context.Context, in *MsgUpdateSnapshotLimit, opts ...grpc.CallOption) (*MsgUpdateSnapshotLimitResponse, error)
	SubmitAttestationEvidence(ctx context.Context, in *MsgSubmitAttestationEvidence, opts ...grpc.CallOption) (*MsgSubmitAttestationEvidenceResponse, error)
	SubmitValsetSignatureEvidence(ctx context.Context, in *MsgSubmitValsetSignatureEvidence, opts ...grpc.CallOption) (*MsgSubmitValsetSignatureEvidenceResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error) {
	out := new(MsgUpdateParamsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/UpdateParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) RequestAttestations(ctx context.Context, in *MsgRequestAttestations, opts ...grpc.CallOption) (*MsgRequestAttestationsResponse, error) {
	out := new(MsgRequestAttestationsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/RequestAttestations", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) WithdrawTokens(ctx context.Context, in *MsgWithdrawTokens, opts ...grpc.CallOption) (*MsgWithdrawTokensResponse, error) {
	out := new(MsgWithdrawTokensResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/WithdrawTokens", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ClaimDeposits(ctx context.Context, in *MsgClaimDepositsRequest, opts ...grpc.CallOption) (*MsgClaimDepositsResponse, error) {
	out := new(MsgClaimDepositsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/ClaimDeposits", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UpdateSnapshotLimit(ctx context.Context, in *MsgUpdateSnapshotLimit, opts ...grpc.CallOption) (*MsgUpdateSnapshotLimitResponse, error) {
	out := new(MsgUpdateSnapshotLimitResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/UpdateSnapshotLimit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) SubmitAttestationEvidence(ctx context.Context, in *MsgSubmitAttestationEvidence, opts ...grpc.CallOption) (*MsgSubmitAttestationEvidenceResponse, error) {
	out := new(MsgSubmitAttestationEvidenceResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/SubmitAttestationEvidence", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) SubmitValsetSignatureEvidence(ctx context.Context, in *MsgSubmitValsetSignatureEvidence, opts ...grpc.CallOption) (*MsgSubmitValsetSignatureEvidenceResponse, error) {
	out := new(MsgSubmitValsetSignatureEvidenceResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/SubmitValsetSignatureEvidence", in, out, opts...)
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
	RequestAttestations(context.Context, *MsgRequestAttestations) (*MsgRequestAttestationsResponse, error)
	WithdrawTokens(context.Context, *MsgWithdrawTokens) (*MsgWithdrawTokensResponse, error)
	ClaimDeposits(context.Context, *MsgClaimDepositsRequest) (*MsgClaimDepositsResponse, error)
	UpdateSnapshotLimit(context.Context, *MsgUpdateSnapshotLimit) (*MsgUpdateSnapshotLimitResponse, error)
	SubmitAttestationEvidence(context.Context, *MsgSubmitAttestationEvidence) (*MsgSubmitAttestationEvidenceResponse, error)
	SubmitValsetSignatureEvidence(context.Context, *MsgSubmitValsetSignatureEvidence) (*MsgSubmitValsetSignatureEvidenceResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (UnimplementedMsgServer) RequestAttestations(context.Context, *MsgRequestAttestations) (*MsgRequestAttestationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestAttestations not implemented")
}
func (UnimplementedMsgServer) WithdrawTokens(context.Context, *MsgWithdrawTokens) (*MsgWithdrawTokensResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WithdrawTokens not implemented")
}
func (UnimplementedMsgServer) ClaimDeposits(context.Context, *MsgClaimDepositsRequest) (*MsgClaimDepositsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ClaimDeposits not implemented")
}
func (UnimplementedMsgServer) UpdateSnapshotLimit(context.Context, *MsgUpdateSnapshotLimit) (*MsgUpdateSnapshotLimitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateSnapshotLimit not implemented")
}
func (UnimplementedMsgServer) SubmitAttestationEvidence(context.Context, *MsgSubmitAttestationEvidence) (*MsgSubmitAttestationEvidenceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitAttestationEvidence not implemented")
}
func (UnimplementedMsgServer) SubmitValsetSignatureEvidence(context.Context, *MsgSubmitValsetSignatureEvidence) (*MsgSubmitValsetSignatureEvidenceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitValsetSignatureEvidence not implemented")
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
		FullMethod: "/layer.bridge.Msg/UpdateParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_RequestAttestations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRequestAttestations)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RequestAttestations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Msg/RequestAttestations",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RequestAttestations(ctx, req.(*MsgRequestAttestations))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_WithdrawTokens_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgWithdrawTokens)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).WithdrawTokens(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Msg/WithdrawTokens",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).WithdrawTokens(ctx, req.(*MsgWithdrawTokens))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ClaimDeposits_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgClaimDepositsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ClaimDeposits(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Msg/ClaimDeposits",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ClaimDeposits(ctx, req.(*MsgClaimDepositsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UpdateSnapshotLimit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateSnapshotLimit)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateSnapshotLimit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Msg/UpdateSnapshotLimit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateSnapshotLimit(ctx, req.(*MsgUpdateSnapshotLimit))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SubmitAttestationEvidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitAttestationEvidence)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitAttestationEvidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Msg/SubmitAttestationEvidence",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitAttestationEvidence(ctx, req.(*MsgSubmitAttestationEvidence))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SubmitValsetSignatureEvidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitValsetSignatureEvidence)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitValsetSignatureEvidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.bridge.Msg/SubmitValsetSignatureEvidence",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitValsetSignatureEvidence(ctx, req.(*MsgSubmitValsetSignatureEvidence))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "layer.bridge.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
		{
			MethodName: "RequestAttestations",
			Handler:    _Msg_RequestAttestations_Handler,
		},
		{
			MethodName: "WithdrawTokens",
			Handler:    _Msg_WithdrawTokens_Handler,
		},
		{
			MethodName: "ClaimDeposits",
			Handler:    _Msg_ClaimDeposits_Handler,
		},
		{
			MethodName: "UpdateSnapshotLimit",
			Handler:    _Msg_UpdateSnapshotLimit_Handler,
		},
		{
			MethodName: "SubmitAttestationEvidence",
			Handler:    _Msg_SubmitAttestationEvidence_Handler,
		},
		{
			MethodName: "SubmitValsetSignatureEvidence",
			Handler:    _Msg_SubmitValsetSignatureEvidence_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/bridge/tx.proto",
}
