// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/oracle/aggregate_reporter.proto

package types

import (
	fmt "fmt"
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

type AggregateReporter struct {
	Reporter    string `protobuf:"bytes,1,opt,name=reporter,proto3" json:"reporter,omitempty"`
	Power       int64  `protobuf:"varint,2,opt,name=power,proto3" json:"power,omitempty"`
	BlockNumber int64  `protobuf:"varint,3,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

func (m *AggregateReporter) Reset()         { *m = AggregateReporter{} }
func (m *AggregateReporter) String() string { return proto.CompactTextString(m) }
func (*AggregateReporter) ProtoMessage()    {}
func (*AggregateReporter) Descriptor() ([]byte, []int) {
	return fileDescriptor_c1d4f41be31c27a7, []int{0}
}
func (m *AggregateReporter) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AggregateReporter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AggregateReporter.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AggregateReporter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AggregateReporter.Merge(m, src)
}
func (m *AggregateReporter) XXX_Size() int {
	return m.Size()
}
func (m *AggregateReporter) XXX_DiscardUnknown() {
	xxx_messageInfo_AggregateReporter.DiscardUnknown(m)
}

var xxx_messageInfo_AggregateReporter proto.InternalMessageInfo

func (m *AggregateReporter) GetReporter() string {
	if m != nil {
		return m.Reporter
	}
	return ""
}

func (m *AggregateReporter) GetPower() int64 {
	if m != nil {
		return m.Power
	}
	return 0
}

func (m *AggregateReporter) GetBlockNumber() int64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

func init() {
	proto.RegisterType((*AggregateReporter)(nil), "layer.oracle.AggregateReporter")
}

func init() {
	proto.RegisterFile("layer/oracle/aggregate_reporter.proto", fileDescriptor_c1d4f41be31c27a7)
}

var fileDescriptor_c1d4f41be31c27a7 = []byte{
	// 204 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xcd, 0x49, 0xac, 0x4c,
	0x2d, 0xd2, 0xcf, 0x2f, 0x4a, 0x4c, 0xce, 0x49, 0xd5, 0x4f, 0x4c, 0x4f, 0x2f, 0x4a, 0x4d, 0x4f,
	0x2c, 0x49, 0x8d, 0x2f, 0x4a, 0x2d, 0xc8, 0x2f, 0x2a, 0x49, 0x2d, 0xd2, 0x2b, 0x28, 0xca, 0x2f,
	0xc9, 0x17, 0xe2, 0x01, 0x2b, 0xd3, 0x83, 0x28, 0x53, 0xca, 0xe0, 0x12, 0x74, 0x84, 0xa9, 0x0c,
	0x82, 0x2a, 0x14, 0x92, 0xe2, 0xe2, 0x80, 0x69, 0x92, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0x82,
	0xf3, 0x85, 0x44, 0xb8, 0x58, 0x0b, 0xf2, 0xcb, 0x53, 0x8b, 0x24, 0x98, 0x14, 0x18, 0x35, 0x98,
	0x83, 0x20, 0x1c, 0x21, 0x45, 0x2e, 0x9e, 0xa4, 0x9c, 0xfc, 0xe4, 0xec, 0xf8, 0xbc, 0xd2, 0xdc,
	0xa4, 0xd4, 0x22, 0x09, 0x66, 0xb0, 0x24, 0x37, 0x58, 0xcc, 0x0f, 0x2c, 0xe4, 0xe4, 0x7c, 0xe2,
	0x91, 0x1c, 0xe3, 0x85, 0x47, 0x72, 0x8c, 0x0f, 0x1e, 0xc9, 0x31, 0x4e, 0x78, 0x2c, 0xc7, 0x70,
	0xe1, 0xb1, 0x1c, 0xc3, 0x8d, 0xc7, 0x72, 0x0c, 0x51, 0x9a, 0xe9, 0x99, 0x25, 0x19, 0xa5, 0x49,
	0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0x25, 0xa9, 0x39, 0x39, 0xf9, 0x45, 0xba, 0x99, 0xf9, 0xfa, 0x10,
	0xdf, 0x54, 0xc0, 0xfc, 0x53, 0x52, 0x59, 0x90, 0x5a, 0x9c, 0xc4, 0x06, 0xf6, 0x83, 0x31, 0x20,
	0x00, 0x00, 0xff, 0xff, 0x0b, 0x5c, 0xd7, 0xdb, 0xec, 0x00, 0x00, 0x00,
}

func (m *AggregateReporter) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AggregateReporter) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AggregateReporter) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.BlockNumber != 0 {
		i = encodeVarintAggregateReporter(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x18
	}
	if m.Power != 0 {
		i = encodeVarintAggregateReporter(dAtA, i, uint64(m.Power))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Reporter) > 0 {
		i -= len(m.Reporter)
		copy(dAtA[i:], m.Reporter)
		i = encodeVarintAggregateReporter(dAtA, i, uint64(len(m.Reporter)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintAggregateReporter(dAtA []byte, offset int, v uint64) int {
	offset -= sovAggregateReporter(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *AggregateReporter) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Reporter)
	if l > 0 {
		n += 1 + l + sovAggregateReporter(uint64(l))
	}
	if m.Power != 0 {
		n += 1 + sovAggregateReporter(uint64(m.Power))
	}
	if m.BlockNumber != 0 {
		n += 1 + sovAggregateReporter(uint64(m.BlockNumber))
	}
	return n
}

func sovAggregateReporter(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAggregateReporter(x uint64) (n int) {
	return sovAggregateReporter(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *AggregateReporter) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAggregateReporter
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
			return fmt.Errorf("proto: AggregateReporter: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AggregateReporter: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reporter", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregateReporter
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
				return ErrInvalidLengthAggregateReporter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAggregateReporter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Reporter = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Power", wireType)
			}
			m.Power = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregateReporter
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Power |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockNumber", wireType)
			}
			m.BlockNumber = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregateReporter
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BlockNumber |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipAggregateReporter(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAggregateReporter
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
func skipAggregateReporter(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAggregateReporter
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
					return 0, ErrIntOverflowAggregateReporter
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
					return 0, ErrIntOverflowAggregateReporter
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
				return 0, ErrInvalidLengthAggregateReporter
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAggregateReporter
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAggregateReporter
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAggregateReporter        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAggregateReporter          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAggregateReporter = fmt.Errorf("proto: unexpected end of group")
)
