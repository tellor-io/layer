// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/registry/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// MsgRegisterSpec defines the Msg/RegisterSpec request type.
type MsgRegisterSpec struct {
	// address that registers the data spec
	Registrar string `protobuf:"bytes,1,opt,name=registrar,proto3" json:"registrar,omitempty"`
	// name of the query type (ie. "SpotPrice")
	QueryType string `protobuf:"bytes,2,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
	// data spec
	Spec DataSpec `protobuf:"bytes,3,opt,name=spec,proto3" json:"spec"`
}

func (m *MsgRegisterSpec) Reset()         { *m = MsgRegisterSpec{} }
func (m *MsgRegisterSpec) String() string { return proto.CompactTextString(m) }
func (*MsgRegisterSpec) ProtoMessage()    {}
func (*MsgRegisterSpec) Descriptor() ([]byte, []int) {
	return fileDescriptor_6dfd681be11a64dd, []int{0}
}
func (m *MsgRegisterSpec) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRegisterSpec) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRegisterSpec.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRegisterSpec) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRegisterSpec.Merge(m, src)
}
func (m *MsgRegisterSpec) XXX_Size() int {
	return m.Size()
}
func (m *MsgRegisterSpec) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRegisterSpec.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRegisterSpec proto.InternalMessageInfo

func (m *MsgRegisterSpec) GetRegistrar() string {
	if m != nil {
		return m.Registrar
	}
	return ""
}

func (m *MsgRegisterSpec) GetQueryType() string {
	if m != nil {
		return m.QueryType
	}
	return ""
}

func (m *MsgRegisterSpec) GetSpec() DataSpec {
	if m != nil {
		return m.Spec
	}
	return DataSpec{}
}

// MsgRegisterSpecResponse defines the Msg/RegisterSpec response type.
type MsgRegisterSpecResponse struct {
}

func (m *MsgRegisterSpecResponse) Reset()         { *m = MsgRegisterSpecResponse{} }
func (m *MsgRegisterSpecResponse) String() string { return proto.CompactTextString(m) }
func (*MsgRegisterSpecResponse) ProtoMessage()    {}
func (*MsgRegisterSpecResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_6dfd681be11a64dd, []int{1}
}
func (m *MsgRegisterSpecResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRegisterSpecResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRegisterSpecResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRegisterSpecResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRegisterSpecResponse.Merge(m, src)
}
func (m *MsgRegisterSpecResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgRegisterSpecResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRegisterSpecResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRegisterSpecResponse proto.InternalMessageInfo

// MsgUpdateDataSpec is the Msg/UpdateDataSpec request type.
type MsgUpdateDataSpec struct {
	// authority is the address that is allowed calling this msg.
	Authority string `protobuf:"bytes,1,opt,name=authority,proto3" json:"authority,omitempty"`
	// query type to update
	QueryType string `protobuf:"bytes,2,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
	// data spec update
	Spec DataSpec `protobuf:"bytes,3,opt,name=spec,proto3" json:"spec"`
}

func (m *MsgUpdateDataSpec) Reset()         { *m = MsgUpdateDataSpec{} }
func (m *MsgUpdateDataSpec) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateDataSpec) ProtoMessage()    {}
func (*MsgUpdateDataSpec) Descriptor() ([]byte, []int) {
	return fileDescriptor_6dfd681be11a64dd, []int{2}
}
func (m *MsgUpdateDataSpec) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateDataSpec) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateDataSpec.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateDataSpec) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateDataSpec.Merge(m, src)
}
func (m *MsgUpdateDataSpec) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateDataSpec) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateDataSpec.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateDataSpec proto.InternalMessageInfo

func (m *MsgUpdateDataSpec) GetAuthority() string {
	if m != nil {
		return m.Authority
	}
	return ""
}

func (m *MsgUpdateDataSpec) GetQueryType() string {
	if m != nil {
		return m.QueryType
	}
	return ""
}

func (m *MsgUpdateDataSpec) GetSpec() DataSpec {
	if m != nil {
		return m.Spec
	}
	return DataSpec{}
}

type MsgUpdateDataSpecResponse struct {
}

func (m *MsgUpdateDataSpecResponse) Reset()         { *m = MsgUpdateDataSpecResponse{} }
func (m *MsgUpdateDataSpecResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateDataSpecResponse) ProtoMessage()    {}
func (*MsgUpdateDataSpecResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_6dfd681be11a64dd, []int{3}
}
func (m *MsgUpdateDataSpecResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateDataSpecResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateDataSpecResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateDataSpecResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateDataSpecResponse.Merge(m, src)
}
func (m *MsgUpdateDataSpecResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateDataSpecResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateDataSpecResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateDataSpecResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgRegisterSpec)(nil), "layer.registry.MsgRegisterSpec")
	proto.RegisterType((*MsgRegisterSpecResponse)(nil), "layer.registry.MsgRegisterSpecResponse")
	proto.RegisterType((*MsgUpdateDataSpec)(nil), "layer.registry.MsgUpdateDataSpec")
	proto.RegisterType((*MsgUpdateDataSpecResponse)(nil), "layer.registry.MsgUpdateDataSpecResponse")
}

func init() { proto.RegisterFile("layer/registry/tx.proto", fileDescriptor_6dfd681be11a64dd) }

var fileDescriptor_6dfd681be11a64dd = []byte{
	// 433 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x52, 0xb1, 0xef, 0xd2, 0x40,
	0x18, 0xed, 0x09, 0x9a, 0x70, 0x1a, 0x0c, 0x0d, 0x09, 0xa5, 0xc6, 0x82, 0x5d, 0x44, 0x0c, 0xbd,
	0x88, 0xd1, 0xc1, 0x4d, 0xa2, 0x23, 0x4b, 0xd1, 0xc4, 0x38, 0x48, 0x8e, 0xf6, 0x72, 0x34, 0xa1,
	0x5c, 0xbd, 0x3b, 0x0c, 0xdd, 0x8c, 0xa3, 0x93, 0x7f, 0x88, 0x03, 0x83, 0x7f, 0x83, 0x61, 0x24,
	0x26, 0x26, 0x4e, 0xc6, 0xc0, 0xc0, 0xbf, 0x61, 0x7a, 0x6d, 0x21, 0x94, 0x5f, 0xc2, 0xf4, 0x5b,
	0xda, 0xde, 0xf7, 0xde, 0x7d, 0xef, 0x7b, 0xdf, 0x2b, 0x6c, 0xcc, 0x70, 0x4c, 0x38, 0xe2, 0x84,
	0x06, 0x42, 0xf2, 0x18, 0xc9, 0xa5, 0x13, 0x71, 0x26, 0x99, 0x5e, 0x55, 0x80, 0x93, 0x03, 0x66,
	0x0d, 0x87, 0xc1, 0x9c, 0x21, 0xf5, 0x4c, 0x29, 0x66, 0xc3, 0x63, 0x22, 0x64, 0x02, 0x85, 0x82,
	0xa2, 0x4f, 0x4f, 0x92, 0x57, 0x06, 0x34, 0x53, 0x60, 0xac, 0x4e, 0x28, 0x3d, 0x64, 0x50, 0x9d,
	0x32, 0xca, 0xd2, 0x7a, 0xf2, 0x95, 0x55, 0xad, 0xc2, 0x14, 0x3e, 0x96, 0x78, 0x2c, 0x22, 0xe2,
	0xa5, 0xb8, 0xfd, 0x1d, 0xc0, 0xbb, 0x43, 0x41, 0x5d, 0x85, 0x13, 0x3e, 0x8a, 0x88, 0xa7, 0x3f,
	0x87, 0x95, 0x8c, 0x8f, 0xb9, 0x01, 0xda, 0xa0, 0x53, 0x19, 0x18, 0xbf, 0x7e, 0xf4, 0xea, 0x99,
	0xdc, 0x4b, 0xdf, 0xe7, 0x44, 0x88, 0x91, 0xe4, 0xc1, 0x9c, 0xba, 0x47, 0xaa, 0x7e, 0x1f, 0xc2,
	0x8f, 0x0b, 0xc2, 0xe3, 0xb1, 0x8c, 0x23, 0x62, 0xdc, 0x48, 0x2e, 0xba, 0x15, 0x55, 0x79, 0x13,
	0x47, 0x44, 0xef, 0xc3, 0x72, 0x22, 0x6c, 0x94, 0xda, 0xa0, 0x73, 0xbb, 0x6f, 0x38, 0xa7, 0x6b,
	0x70, 0x5e, 0x61, 0x89, 0x13, 0xf9, 0x41, 0x79, 0xfd, 0xb7, 0xa5, 0xb9, 0x8a, 0xfb, 0xa2, 0xfa,
	0x65, 0xbf, 0xea, 0x1e, 0x25, 0xec, 0x26, 0x6c, 0x14, 0xa6, 0x75, 0x89, 0x88, 0xd8, 0x5c, 0x10,
	0xfb, 0x37, 0x80, 0xb5, 0xa1, 0xa0, 0x6f, 0x23, 0x1f, 0x4b, 0x92, 0x37, 0x4b, 0xbc, 0xe0, 0x85,
	0x9c, 0x32, 0x1e, 0xc8, 0xf8, 0xb2, 0x97, 0x03, 0xf5, 0x3a, 0xbc, 0x3c, 0x53, 0x5e, 0x0e, 0x12,
	0x5f, 0xf7, 0xab, 0xae, 0x9d, 0xa6, 0xb3, 0x3c, 0xe6, 0x73, 0xe6, 0xc0, 0xbe, 0x07, 0x9b, 0x67,
	0xc5, 0xdc, 0x74, 0xff, 0x27, 0x80, 0xa5, 0xa1, 0xa0, 0xfa, 0x3b, 0x78, 0xe7, 0x24, 0xc2, 0x56,
	0x71, 0xa2, 0xc2, 0xd6, 0xcc, 0x87, 0x17, 0x08, 0xb9, 0x82, 0xfe, 0x01, 0x56, 0x0b, 0x2b, 0x7d,
	0x70, 0xc5, 0xd5, 0x53, 0x8a, 0xf9, 0xe8, 0x22, 0x25, 0xef, 0x6f, 0xde, 0xfc, 0xbc, 0x5f, 0x75,
	0xc1, 0xe0, 0xf5, 0x7a, 0x6b, 0x81, 0xcd, 0xd6, 0x02, 0xff, 0xb6, 0x16, 0xf8, 0xb6, 0xb3, 0xb4,
	0xcd, 0xce, 0xd2, 0xfe, 0xec, 0x2c, 0xed, 0xfd, 0x63, 0x1a, 0xc8, 0xe9, 0x62, 0xe2, 0x78, 0x2c,
	0x44, 0x92, 0xcc, 0x66, 0x8c, 0xf7, 0x02, 0x86, 0xce, 0x16, 0x97, 0xe4, 0x24, 0x26, 0xb7, 0xd4,
	0x5f, 0xfd, 0xf4, 0x7f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x41, 0x01, 0x74, 0x1c, 0x7d, 0x03, 0x00,
	0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// RegisterSpec defines a method for registering a new data specification.
	RegisterSpec(ctx context.Context, in *MsgRegisterSpec, opts ...grpc.CallOption) (*MsgRegisterSpecResponse, error)
	// UpdateDataSpec defines a method for updating an existing data specification.
	UpdateDataSpec(ctx context.Context, in *MsgUpdateDataSpec, opts ...grpc.CallOption) (*MsgUpdateDataSpecResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) RegisterSpec(ctx context.Context, in *MsgRegisterSpec, opts ...grpc.CallOption) (*MsgRegisterSpecResponse, error) {
	out := new(MsgRegisterSpecResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Msg/RegisterSpec", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UpdateDataSpec(ctx context.Context, in *MsgUpdateDataSpec, opts ...grpc.CallOption) (*MsgUpdateDataSpecResponse, error) {
	out := new(MsgUpdateDataSpecResponse)
	err := c.cc.Invoke(ctx, "/layer.registry.Msg/UpdateDataSpec", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// RegisterSpec defines a method for registering a new data specification.
	RegisterSpec(context.Context, *MsgRegisterSpec) (*MsgRegisterSpecResponse, error)
	// UpdateDataSpec defines a method for updating an existing data specification.
	UpdateDataSpec(context.Context, *MsgUpdateDataSpec) (*MsgUpdateDataSpecResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) RegisterSpec(ctx context.Context, req *MsgRegisterSpec) (*MsgRegisterSpecResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterSpec not implemented")
}
func (*UnimplementedMsgServer) UpdateDataSpec(ctx context.Context, req *MsgUpdateDataSpec) (*MsgUpdateDataSpecResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDataSpec not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_RegisterSpec_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterSpec)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterSpec(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Msg/RegisterSpec",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterSpec(ctx, req.(*MsgRegisterSpec))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UpdateDataSpec_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateDataSpec)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateDataSpec(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/layer.registry.Msg/UpdateDataSpec",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateDataSpec(ctx, req.(*MsgUpdateDataSpec))
	}
	return interceptor(ctx, in, info, handler)
}

var Msg_serviceDesc = _Msg_serviceDesc
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "layer.registry.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterSpec",
			Handler:    _Msg_RegisterSpec_Handler,
		},
		{
			MethodName: "UpdateDataSpec",
			Handler:    _Msg_UpdateDataSpec_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/registry/tx.proto",
}

func (m *MsgRegisterSpec) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRegisterSpec) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRegisterSpec) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Spec.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.QueryType) > 0 {
		i -= len(m.QueryType)
		copy(dAtA[i:], m.QueryType)
		i = encodeVarintTx(dAtA, i, uint64(len(m.QueryType)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Registrar) > 0 {
		i -= len(m.Registrar)
		copy(dAtA[i:], m.Registrar)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Registrar)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgRegisterSpecResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRegisterSpecResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRegisterSpecResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgUpdateDataSpec) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateDataSpec) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateDataSpec) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Spec.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.QueryType) > 0 {
		i -= len(m.QueryType)
		copy(dAtA[i:], m.QueryType)
		i = encodeVarintTx(dAtA, i, uint64(len(m.QueryType)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgUpdateDataSpecResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateDataSpecResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateDataSpecResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgRegisterSpec) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Registrar)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.QueryType)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Spec.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgRegisterSpecResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgUpdateDataSpec) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Authority)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.QueryType)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Spec.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgUpdateDataSpecResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgRegisterSpec) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgRegisterSpec: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRegisterSpec: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Registrar", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Registrar = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Spec", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Spec.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgRegisterSpecResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgRegisterSpecResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRegisterSpecResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgUpdateDataSpec) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgUpdateDataSpec: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateDataSpec: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Authority", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Spec", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Spec.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgUpdateDataSpecResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgUpdateDataSpecResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateDataSpecResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
