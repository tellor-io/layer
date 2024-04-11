// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/bridge/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
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

type MsgRequestAttestations struct {
	Creator   string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	QueryId   string `protobuf:"bytes,2,opt,name=queryId,proto3" json:"queryId,omitempty"`
	Timestamp string `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (m *MsgRequestAttestations) Reset()         { *m = MsgRequestAttestations{} }
func (m *MsgRequestAttestations) String() string { return proto.CompactTextString(m) }
func (*MsgRequestAttestations) ProtoMessage()    {}
func (*MsgRequestAttestations) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb4091a52ebadfca, []int{0}
}
func (m *MsgRequestAttestations) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRequestAttestations) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRequestAttestations.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRequestAttestations) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRequestAttestations.Merge(m, src)
}
func (m *MsgRequestAttestations) XXX_Size() int {
	return m.Size()
}
func (m *MsgRequestAttestations) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRequestAttestations.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRequestAttestations proto.InternalMessageInfo

func (m *MsgRequestAttestations) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *MsgRequestAttestations) GetQueryId() string {
	if m != nil {
		return m.QueryId
	}
	return ""
}

func (m *MsgRequestAttestations) GetTimestamp() string {
	if m != nil {
		return m.Timestamp
	}
	return ""
}

type MsgRequestAttestationsResponse struct {
}

func (m *MsgRequestAttestationsResponse) Reset()         { *m = MsgRequestAttestationsResponse{} }
func (m *MsgRequestAttestationsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgRequestAttestationsResponse) ProtoMessage()    {}
func (*MsgRequestAttestationsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb4091a52ebadfca, []int{1}
}
func (m *MsgRequestAttestationsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRequestAttestationsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRequestAttestationsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRequestAttestationsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRequestAttestationsResponse.Merge(m, src)
}
func (m *MsgRequestAttestationsResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgRequestAttestationsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRequestAttestationsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRequestAttestationsResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgRequestAttestations)(nil), "layer.bridge.MsgRequestAttestations")
	proto.RegisterType((*MsgRequestAttestationsResponse)(nil), "layer.bridge.MsgRequestAttestationsResponse")
}

func init() { proto.RegisterFile("layer/bridge/tx.proto", fileDescriptor_cb4091a52ebadfca) }

var fileDescriptor_cb4091a52ebadfca = []byte{
	// 266 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0xcd, 0x49, 0xac, 0x4c,
	0x2d, 0xd2, 0x4f, 0x2a, 0xca, 0x4c, 0x49, 0x4f, 0xd5, 0x2f, 0xa9, 0xd0, 0x2b, 0x28, 0xca, 0x2f,
	0xc9, 0x17, 0xe2, 0x01, 0x0b, 0xeb, 0x41, 0x84, 0xa5, 0xc4, 0x93, 0xf3, 0x8b, 0x73, 0xf3, 0x8b,
	0xf5, 0x73, 0x8b, 0xd3, 0xf5, 0xcb, 0x0c, 0x41, 0x14, 0x44, 0x99, 0x52, 0x05, 0x97, 0x98, 0x6f,
	0x71, 0x7a, 0x50, 0x6a, 0x61, 0x69, 0x6a, 0x71, 0x89, 0x63, 0x49, 0x49, 0x6a, 0x71, 0x49, 0x62,
	0x49, 0x66, 0x7e, 0x5e, 0xb1, 0x90, 0x04, 0x17, 0x7b, 0x72, 0x51, 0x6a, 0x62, 0x49, 0x7e, 0x91,
	0x04, 0xa3, 0x02, 0xa3, 0x06, 0x67, 0x10, 0x8c, 0x0b, 0x92, 0x29, 0x2c, 0x4d, 0x2d, 0xaa, 0xf4,
	0x4c, 0x91, 0x60, 0x82, 0xc8, 0x40, 0xb9, 0x42, 0x32, 0x5c, 0x9c, 0x25, 0x99, 0xb9, 0x20, 0x33,
	0x72, 0x0b, 0x24, 0x98, 0xc1, 0x72, 0x08, 0x01, 0x2b, 0x9e, 0xa6, 0xe7, 0x1b, 0xb4, 0x60, 0xa6,
	0x28, 0x29, 0x70, 0xc9, 0x61, 0xb7, 0x39, 0x28, 0xb5, 0xb8, 0x20, 0x3f, 0xaf, 0x38, 0xd5, 0xa8,
	0x80, 0x8b, 0xd9, 0xb7, 0x38, 0x5d, 0x28, 0x93, 0x4b, 0x18, 0x9b, 0xfb, 0x54, 0xf4, 0x90, 0x7d,
	0xa8, 0x87, 0xdd, 0x2c, 0x29, 0x1d, 0x62, 0x54, 0xc1, 0x6c, 0x74, 0x72, 0x3e, 0xf1, 0x48, 0x8e,
	0xf1, 0xc2, 0x23, 0x39, 0xc6, 0x07, 0x8f, 0xe4, 0x18, 0x27, 0x3c, 0x96, 0x63, 0xb8, 0xf0, 0x58,
	0x8e, 0xe1, 0xc6, 0x63, 0x39, 0x86, 0x28, 0xcd, 0xf4, 0xcc, 0x92, 0x8c, 0xd2, 0x24, 0xbd, 0xe4,
	0xfc, 0x5c, 0xfd, 0x92, 0xd4, 0x9c, 0x9c, 0xfc, 0x22, 0xdd, 0xcc, 0x7c, 0x7d, 0x48, 0xd0, 0x57,
	0xc0, 0x03, 0xbf, 0xb2, 0x20, 0xb5, 0x38, 0x89, 0x0d, 0x1c, 0xb2, 0xc6, 0x80, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xed, 0x91, 0xbe, 0xa8, 0x99, 0x01, 0x00, 0x00,
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
	RequestAttestations(ctx context.Context, in *MsgRequestAttestations, opts ...grpc.CallOption) (*MsgRequestAttestationsResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) RequestAttestations(ctx context.Context, in *MsgRequestAttestations, opts ...grpc.CallOption) (*MsgRequestAttestationsResponse, error) {
	out := new(MsgRequestAttestationsResponse)
	err := c.cc.Invoke(ctx, "/layer.bridge.Msg/RequestAttestations", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	RequestAttestations(context.Context, *MsgRequestAttestations) (*MsgRequestAttestationsResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) RequestAttestations(ctx context.Context, req *MsgRequestAttestations) (*MsgRequestAttestationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestAttestations not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
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

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "layer.bridge.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RequestAttestations",
			Handler:    _Msg_RequestAttestations_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "layer/bridge/tx.proto",
}

func (m *MsgRequestAttestations) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRequestAttestations) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRequestAttestations) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Timestamp) > 0 {
		i -= len(m.Timestamp)
		copy(dAtA[i:], m.Timestamp)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Timestamp)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.QueryId) > 0 {
		i -= len(m.QueryId)
		copy(dAtA[i:], m.QueryId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.QueryId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgRequestAttestationsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRequestAttestationsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRequestAttestationsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
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
func (m *MsgRequestAttestations) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.QueryId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Timestamp)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgRequestAttestationsResponse) Size() (n int) {
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
func (m *MsgRequestAttestations) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgRequestAttestations: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRequestAttestations: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
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
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryId", wireType)
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
			m.QueryId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
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
			m.Timestamp = string(dAtA[iNdEx:postIndex])
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
func (m *MsgRequestAttestationsResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgRequestAttestationsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRequestAttestationsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
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
