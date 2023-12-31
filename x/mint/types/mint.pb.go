// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/mint/mint.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
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

// Minter represents the mint state.
type Minter struct {
	// PreviousBlockTime is the timestamp of the previous block.
	PreviousBlockTime *time.Time `protobuf:"bytes,1,opt,name=previous_block_time,json=previousBlockTime,proto3,stdtime" json:"previous_block_time,omitempty"`
	// BondDenom is the denomination of the token that should be minted.
	BondDenom string `protobuf:"bytes,2,opt,name=bond_denom,json=bondDenom,proto3" json:"bond_denom,omitempty"`
}

func (m *Minter) Reset()         { *m = Minter{} }
func (m *Minter) String() string { return proto.CompactTextString(m) }
func (*Minter) ProtoMessage()    {}
func (*Minter) Descriptor() ([]byte, []int) {
	return fileDescriptor_814b376460bcd2cc, []int{0}
}
func (m *Minter) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Minter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Minter.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Minter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Minter.Merge(m, src)
}
func (m *Minter) XXX_Size() int {
	return m.Size()
}
func (m *Minter) XXX_DiscardUnknown() {
	xxx_messageInfo_Minter.DiscardUnknown(m)
}

var xxx_messageInfo_Minter proto.InternalMessageInfo

func (m *Minter) GetPreviousBlockTime() *time.Time {
	if m != nil {
		return m.PreviousBlockTime
	}
	return nil
}

func (m *Minter) GetBondDenom() string {
	if m != nil {
		return m.BondDenom
	}
	return ""
}

func init() {
	proto.RegisterType((*Minter)(nil), "layer.mint.Minter")
}

func init() { proto.RegisterFile("layer/mint/mint.proto", fileDescriptor_814b376460bcd2cc) }

var fileDescriptor_814b376460bcd2cc = []byte{
	// 258 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x34, 0x90, 0xc1, 0x4a, 0xf4, 0x30,
	0x14, 0x85, 0x9b, 0x9f, 0x9f, 0x81, 0x89, 0x2b, 0xab, 0xc2, 0x58, 0x30, 0x33, 0xb8, 0x71, 0x36,
	0x26, 0xa0, 0x4f, 0x60, 0x71, 0x2b, 0xc8, 0xe0, 0xca, 0x4d, 0x99, 0x74, 0x62, 0x0c, 0x26, 0xbd,
	0x25, 0x49, 0xc5, 0xbe, 0xc5, 0x3c, 0x96, 0xcb, 0x59, 0xba, 0x53, 0xda, 0x17, 0x91, 0x24, 0x76,
	0x13, 0x72, 0xbf, 0x73, 0xee, 0xe1, 0x72, 0xf0, 0x99, 0xde, 0xf6, 0xc2, 0x32, 0xa3, 0x1a, 0x1f,
	0x1f, 0xda, 0x5a, 0xf0, 0x90, 0xe3, 0x88, 0x69, 0x20, 0xc5, 0xa9, 0x04, 0x09, 0x11, 0xb3, 0xf0,
	0x4b, 0x8e, 0xe2, 0xbc, 0x06, 0x67, 0xc0, 0x55, 0x49, 0x48, 0xc3, 0x9f, 0xb4, 0x94, 0x00, 0x52,
	0x0b, 0x16, 0x27, 0xde, 0xbd, 0x30, 0xaf, 0x8c, 0x70, 0x7e, 0x6b, 0xda, 0x64, 0xb8, 0xec, 0xf1,
	0xec, 0x41, 0x35, 0x5e, 0xd8, 0xfc, 0x11, 0x9f, 0xb4, 0x56, 0xbc, 0x2b, 0xe8, 0x5c, 0xc5, 0x35,
	0xd4, 0x6f, 0x55, 0xf0, 0x2e, 0xd0, 0x0a, 0xad, 0x8f, 0x6e, 0x0a, 0x9a, 0x82, 0xe8, 0x14, 0x44,
	0x9f, 0xa6, 0xa0, 0xf2, 0xff, 0xfe, 0x7b, 0x89, 0x36, 0xc7, 0xd3, 0x72, 0x19, 0x76, 0x83, 0x9a,
	0x5f, 0x60, 0xcc, 0xa1, 0xd9, 0x55, 0x3b, 0xd1, 0x80, 0x59, 0xfc, 0x5b, 0xa1, 0xf5, 0x7c, 0x33,
	0x0f, 0xe4, 0x3e, 0x80, 0xf2, 0xee, 0x73, 0x20, 0xe8, 0x30, 0x10, 0xf4, 0x33, 0x10, 0xb4, 0x1f,
	0x49, 0x76, 0x18, 0x49, 0xf6, 0x35, 0x92, 0xec, 0xf9, 0x4a, 0x2a, 0xff, 0xda, 0x71, 0x5a, 0x83,
	0x61, 0x5e, 0x68, 0x0d, 0xf6, 0x5a, 0x01, 0x4b, 0xf5, 0x7c, 0xa4, 0x82, 0x7c, 0xdf, 0x0a, 0xc7,
	0x67, 0xf1, 0x9c, 0xdb, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x12, 0x40, 0xed, 0xa0, 0x3b, 0x01,
	0x00, 0x00,
}

func (m *Minter) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Minter) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Minter) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.BondDenom) > 0 {
		i -= len(m.BondDenom)
		copy(dAtA[i:], m.BondDenom)
		i = encodeVarintMint(dAtA, i, uint64(len(m.BondDenom)))
		i--
		dAtA[i] = 0x12
	}
	if m.PreviousBlockTime != nil {
		n1, err1 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(*m.PreviousBlockTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(*m.PreviousBlockTime):])
		if err1 != nil {
			return 0, err1
		}
		i -= n1
		i = encodeVarintMint(dAtA, i, uint64(n1))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMint(dAtA []byte, offset int, v uint64) int {
	offset -= sovMint(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Minter) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.PreviousBlockTime != nil {
		l = github_com_cosmos_gogoproto_types.SizeOfStdTime(*m.PreviousBlockTime)
		n += 1 + l + sovMint(uint64(l))
	}
	l = len(m.BondDenom)
	if l > 0 {
		n += 1 + l + sovMint(uint64(l))
	}
	return n
}

func sovMint(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMint(x uint64) (n int) {
	return sovMint(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Minter) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMint
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
			return fmt.Errorf("proto: Minter: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Minter: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PreviousBlockTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMint
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
				return ErrInvalidLengthMint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.PreviousBlockTime == nil {
				m.PreviousBlockTime = new(time.Time)
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(m.PreviousBlockTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BondDenom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMint
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
				return ErrInvalidLengthMint
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BondDenom = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMint(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMint
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
func skipMint(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMint
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
					return 0, ErrIntOverflowMint
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
					return 0, ErrIntOverflowMint
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
				return 0, ErrInvalidLengthMint
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMint
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMint
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMint        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMint          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMint = fmt.Errorf("proto: unexpected end of group")
)
