// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/reporter/selection.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Selection is a type that represents a  delegator's selection
type Selection struct {
	// reporter is the address of the reporter being delegated to
	Reporter []byte `protobuf:"bytes,1,opt,name=reporter,proto3" json:"reporter,omitempty"`
	// locked_until_time is the time until which the tokens are locked before they
	// can be used for reporting again
	LockedUntilTime  time.Time `protobuf:"bytes,2,opt,name=locked_until_time,json=lockedUntilTime,proto3,stdtime" json:"locked_until_time"`
	DelegationsCount int64     `protobuf:"varint,3,opt,name=delegations_count,json=delegationsCount,proto3" json:"delegations_count,omitempty"`
}

func (m *Selection) Reset()         { *m = Selection{} }
func (m *Selection) String() string { return proto.CompactTextString(m) }
func (*Selection) ProtoMessage()    {}
func (*Selection) Descriptor() ([]byte, []int) {
	return fileDescriptor_0b0e998201c9cd64, []int{0}
}
func (m *Selection) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Selection) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Selection.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Selection) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Selection.Merge(m, src)
}
func (m *Selection) XXX_Size() int {
	return m.Size()
}
func (m *Selection) XXX_DiscardUnknown() {
	xxx_messageInfo_Selection.DiscardUnknown(m)
}

var xxx_messageInfo_Selection proto.InternalMessageInfo

func (m *Selection) GetReporter() []byte {
	if m != nil {
		return m.Reporter
	}
	return nil
}

func (m *Selection) GetLockedUntilTime() time.Time {
	if m != nil {
		return m.LockedUntilTime
	}
	return time.Time{}
}

func (m *Selection) GetDelegationsCount() int64 {
	if m != nil {
		return m.DelegationsCount
	}
	return 0
}

func init() {
	proto.RegisterType((*Selection)(nil), "layer.reporter.Selection")
}

func init() { proto.RegisterFile("layer/reporter/selection.proto", fileDescriptor_0b0e998201c9cd64) }

var fileDescriptor_0b0e998201c9cd64 = []byte{
	// 296 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x90, 0xb1, 0x4e, 0xc3, 0x30,
	0x10, 0x86, 0x63, 0x2a, 0xa1, 0x12, 0x10, 0xd0, 0x88, 0xa1, 0x64, 0x70, 0x2b, 0xa6, 0x4a, 0x15,
	0xb6, 0x04, 0x6f, 0x50, 0xc4, 0x8e, 0x0a, 0x2c, 0x2c, 0x51, 0x9a, 0x1e, 0xc6, 0xc2, 0xc9, 0x45,
	0xb6, 0x23, 0xd1, 0xb7, 0xe8, 0x53, 0xf0, 0x2c, 0x1d, 0x3b, 0x32, 0x01, 0x4a, 0x5e, 0x04, 0x39,
	0x6e, 0x80, 0xc5, 0xf2, 0xff, 0xff, 0x77, 0xf7, 0x9d, 0x2e, 0xa4, 0x2a, 0x5d, 0x81, 0xe6, 0x1a,
	0x4a, 0xd4, 0x16, 0x34, 0x37, 0xa0, 0x20, 0xb3, 0x12, 0x0b, 0x56, 0x6a, 0xb4, 0x18, 0x1d, 0xb7,
	0x39, 0xeb, 0xf2, 0x78, 0x90, 0xe6, 0xb2, 0x40, 0xde, 0xbe, 0xbe, 0x24, 0x3e, 0xcf, 0xd0, 0xe4,
	0x68, 0x92, 0x56, 0x71, 0x2f, 0x76, 0xd1, 0x99, 0x40, 0x81, 0xde, 0x77, 0xbf, 0x9d, 0x3b, 0x12,
	0x88, 0x42, 0x01, 0x6f, 0xd5, 0xa2, 0x7a, 0xe6, 0x56, 0xe6, 0x60, 0x6c, 0x9a, 0x97, 0xbe, 0xe0,
	0xe2, 0x9d, 0x84, 0x07, 0xf7, 0xdd, 0x22, 0x51, 0x1c, 0xf6, 0x3b, 0xfc, 0x90, 0x8c, 0xc9, 0xe4,
	0x68, 0xfe, 0xab, 0xa3, 0xbb, 0x70, 0xa0, 0x30, 0x7b, 0x85, 0x65, 0x52, 0x15, 0x56, 0xaa, 0xc4,
	0x4d, 0x1a, 0xee, 0x8d, 0xc9, 0xe4, 0xf0, 0x2a, 0x66, 0x1e, 0xc3, 0x3a, 0x0c, 0x7b, 0xe8, 0x30,
	0xb3, 0xfe, 0xe6, 0x73, 0x14, 0xac, 0xbf, 0x46, 0x64, 0x7e, 0xe2, 0xdb, 0x1f, 0x5d, 0xb7, 0xcb,
	0xa3, 0x69, 0x38, 0x58, 0x82, 0x02, 0x91, 0x3a, 0xb6, 0x49, 0x32, 0xac, 0x0a, 0x3b, 0xec, 0x8d,
	0xc9, 0xa4, 0x37, 0x3f, 0xfd, 0x17, 0xdc, 0x38, 0x7f, 0x76, 0xbb, 0xa9, 0x29, 0xd9, 0xd6, 0x94,
	0x7c, 0xd7, 0x94, 0xac, 0x1b, 0x1a, 0x6c, 0x1b, 0x1a, 0x7c, 0x34, 0x34, 0x78, 0x9a, 0x0a, 0x69,
	0x5f, 0xaa, 0x05, 0xcb, 0x30, 0xe7, 0x16, 0x94, 0x42, 0x7d, 0x29, 0x91, 0xfb, 0x63, 0xbf, 0xfd,
	0x9d, 0xdb, 0xae, 0x4a, 0x30, 0x8b, 0xfd, 0x76, 0xc5, 0xeb, 0x9f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0xd7, 0x58, 0x48, 0x59, 0x8d, 0x01, 0x00, 0x00,
}

func (m *Selection) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Selection) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Selection) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.DelegationsCount != 0 {
		i = encodeVarintSelection(dAtA, i, uint64(m.DelegationsCount))
		i--
		dAtA[i] = 0x18
	}
	n1, err1 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.LockedUntilTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.LockedUntilTime):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintSelection(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x12
	if len(m.Reporter) > 0 {
		i -= len(m.Reporter)
		copy(dAtA[i:], m.Reporter)
		i = encodeVarintSelection(dAtA, i, uint64(len(m.Reporter)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintSelection(dAtA []byte, offset int, v uint64) int {
	offset -= sovSelection(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Selection) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Reporter)
	if l > 0 {
		n += 1 + l + sovSelection(uint64(l))
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdTime(m.LockedUntilTime)
	n += 1 + l + sovSelection(uint64(l))
	if m.DelegationsCount != 0 {
		n += 1 + sovSelection(uint64(m.DelegationsCount))
	}
	return n
}

func sovSelection(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSelection(x uint64) (n int) {
	return sovSelection(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Selection) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSelection
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
			return fmt.Errorf("proto: Selection: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Selection: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reporter", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSelection
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSelection
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSelection
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Reporter = append(m.Reporter[:0], dAtA[iNdEx:postIndex]...)
			if m.Reporter == nil {
				m.Reporter = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LockedUntilTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSelection
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
				return ErrInvalidLengthSelection
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSelection
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&m.LockedUntilTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DelegationsCount", wireType)
			}
			m.DelegationsCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSelection
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DelegationsCount |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSelection(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSelection
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
func skipSelection(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSelection
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
					return 0, ErrIntOverflowSelection
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
					return 0, ErrIntOverflowSelection
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
				return 0, ErrInvalidLengthSelection
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSelection
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSelection
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSelection        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSelection          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSelection = fmt.Errorf("proto: unexpected end of group")
)
