// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/oracle/user_tip.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

type UserTipTotal struct {
	Address string     `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Total   types.Coin `protobuf:"bytes,2,opt,name=total,proto3" json:"total"`
}

func (m *UserTipTotal) Reset()         { *m = UserTipTotal{} }
func (m *UserTipTotal) String() string { return proto.CompactTextString(m) }
func (*UserTipTotal) ProtoMessage()    {}
func (*UserTipTotal) Descriptor() ([]byte, []int) {
	return fileDescriptor_aa4312bb1dc0313d, []int{0}
}
func (m *UserTipTotal) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *UserTipTotal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_UserTipTotal.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *UserTipTotal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UserTipTotal.Merge(m, src)
}
func (m *UserTipTotal) XXX_Size() int {
	return m.Size()
}
func (m *UserTipTotal) XXX_DiscardUnknown() {
	xxx_messageInfo_UserTipTotal.DiscardUnknown(m)
}

var xxx_messageInfo_UserTipTotal proto.InternalMessageInfo

func (m *UserTipTotal) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *UserTipTotal) GetTotal() types.Coin {
	if m != nil {
		return m.Total
	}
	return types.Coin{}
}

func init() {
	proto.RegisterType((*UserTipTotal)(nil), "layer.oracle.UserTipTotal")
}

func init() { proto.RegisterFile("layer/oracle/user_tip.proto", fileDescriptor_aa4312bb1dc0313d) }

var fileDescriptor_aa4312bb1dc0313d = []byte{
	// 265 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x34, 0x90, 0x31, 0x4e, 0xf3, 0x30,
	0x18, 0x86, 0xe3, 0x5f, 0x3f, 0x20, 0x42, 0xa7, 0xa8, 0x43, 0x5a, 0x24, 0x53, 0x31, 0x95, 0xa1,
	0xb6, 0x5a, 0xc4, 0x01, 0x48, 0x6f, 0x50, 0xca, 0xc2, 0x52, 0x39, 0xa9, 0x15, 0x2c, 0xb9, 0xf9,
	0x22, 0xfb, 0x2b, 0x22, 0xb7, 0xe0, 0x30, 0x1c, 0xa2, 0x63, 0xc5, 0xc4, 0x84, 0x50, 0x72, 0x11,
	0x94, 0x7c, 0xe9, 0xe6, 0x57, 0xcf, 0x63, 0xbf, 0xf2, 0x1b, 0x5e, 0x5b, 0x55, 0x69, 0x27, 0xc1,
	0xa9, 0xcc, 0x6a, 0xb9, 0xf7, 0xda, 0x6d, 0xd0, 0x94, 0xa2, 0x74, 0x80, 0x10, 0x0d, 0x3a, 0x28,
	0x08, 0x8e, 0x87, 0x39, 0xe4, 0xd0, 0x01, 0xd9, 0x9e, 0xc8, 0x19, 0x8f, 0x32, 0xf0, 0x3b, 0xf0,
	0x1b, 0x02, 0x14, 0x7a, 0xc4, 0x29, 0xc9, 0x54, 0x79, 0x2d, 0xdf, 0xe6, 0xa9, 0x46, 0x35, 0x97,
	0x19, 0x98, 0x82, 0xf8, 0x6d, 0x15, 0x0e, 0x9e, 0xbd, 0x76, 0x6b, 0x53, 0xae, 0x01, 0x95, 0x8d,
	0x16, 0xe1, 0x85, 0xda, 0x6e, 0x9d, 0xf6, 0x3e, 0x66, 0x13, 0x36, 0xbd, 0x4c, 0xe2, 0xaf, 0xcf,
	0xd9, 0xb0, 0x7f, 0xf2, 0x91, 0xc8, 0x13, 0x3a, 0x53, 0xe4, 0xab, 0x93, 0x18, 0x3d, 0x84, 0x67,
	0xd8, 0x5e, 0x8e, 0xff, 0x4d, 0xd8, 0xf4, 0x6a, 0x31, 0x12, 0xbd, 0xde, 0x76, 0x8a, 0xbe, 0x53,
	0x2c, 0xc1, 0x14, 0xc9, 0xff, 0xc3, 0xcf, 0x4d, 0xb0, 0x22, 0x3b, 0x59, 0x1e, 0x6a, 0xce, 0x8e,
	0x35, 0x67, 0xbf, 0x35, 0x67, 0x1f, 0x0d, 0x0f, 0x8e, 0x0d, 0x0f, 0xbe, 0x1b, 0x1e, 0xbc, 0xdc,
	0xe5, 0x06, 0x5f, 0xf7, 0xa9, 0xc8, 0x60, 0x27, 0x51, 0x5b, 0x0b, 0x6e, 0x66, 0x40, 0xd2, 0x4a,
	0xef, 0xa7, 0x9d, 0xb0, 0x2a, 0xb5, 0x4f, 0xcf, 0xbb, 0x6f, 0xdc, 0xff, 0x05, 0x00, 0x00, 0xff,
	0xff, 0xb9, 0xd0, 0xe0, 0xa4, 0x44, 0x01, 0x00, 0x00,
}

func (m *UserTipTotal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *UserTipTotal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *UserTipTotal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Total.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintUserTip(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintUserTip(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintUserTip(dAtA []byte, offset int, v uint64) int {
	offset -= sovUserTip(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *UserTipTotal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovUserTip(uint64(l))
	}
	l = m.Total.Size()
	n += 1 + l + sovUserTip(uint64(l))
	return n
}

func sovUserTip(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozUserTip(x uint64) (n int) {
	return sovUserTip(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *UserTipTotal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUserTip
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
			return fmt.Errorf("proto: UserTipTotal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: UserTipTotal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUserTip
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
				return ErrInvalidLengthUserTip
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthUserTip
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Total", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUserTip
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
				return ErrInvalidLengthUserTip
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthUserTip
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Total.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipUserTip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUserTip
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
func skipUserTip(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowUserTip
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
					return 0, ErrIntOverflowUserTip
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
					return 0, ErrIntOverflowUserTip
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
				return 0, ErrInvalidLengthUserTip
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupUserTip
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthUserTip
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthUserTip        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowUserTip          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupUserTip = fmt.Errorf("proto: unexpected end of group")
)
