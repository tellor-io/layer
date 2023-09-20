// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/registry/data_spec.proto

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

type DataSpec struct {
	DocumentHash string `protobuf:"bytes,1,opt,name=documentHash,proto3" json:"documentHash,omitempty"`
	ValueType    string `protobuf:"bytes,2,opt,name=valueType,proto3" json:"valueType,omitempty"`
}

func (m *DataSpec) Reset()         { *m = DataSpec{} }
func (m *DataSpec) String() string { return proto.CompactTextString(m) }
func (*DataSpec) ProtoMessage()    {}
func (*DataSpec) Descriptor() ([]byte, []int) {
	return fileDescriptor_8c1d9edbb99f1378, []int{0}
}
func (m *DataSpec) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DataSpec) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DataSpec.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DataSpec) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DataSpec.Merge(m, src)
}
func (m *DataSpec) XXX_Size() int {
	return m.Size()
}
func (m *DataSpec) XXX_DiscardUnknown() {
	xxx_messageInfo_DataSpec.DiscardUnknown(m)
}

var xxx_messageInfo_DataSpec proto.InternalMessageInfo

func (m *DataSpec) GetDocumentHash() string {
	if m != nil {
		return m.DocumentHash
	}
	return ""
}

func (m *DataSpec) GetValueType() string {
	if m != nil {
		return m.ValueType
	}
	return ""
}

func init() {
	proto.RegisterType((*DataSpec)(nil), "layer.registry.DataSpec")
}

func init() { proto.RegisterFile("layer/registry/data_spec.proto", fileDescriptor_8c1d9edbb99f1378) }

var fileDescriptor_8c1d9edbb99f1378 = []byte{
	// 164 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xcb, 0x49, 0xac, 0x4c,
	0x2d, 0xd2, 0x2f, 0x4a, 0x4d, 0xcf, 0x2c, 0x2e, 0x29, 0xaa, 0xd4, 0x4f, 0x49, 0x2c, 0x49, 0x8c,
	0x2f, 0x2e, 0x48, 0x4d, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x03, 0xcb, 0xeb, 0xc1,
	0xe4, 0x95, 0x7c, 0xb8, 0x38, 0x5c, 0x12, 0x4b, 0x12, 0x83, 0x0b, 0x52, 0x93, 0x85, 0x94, 0xb8,
	0x78, 0x52, 0xf2, 0x93, 0x4b, 0x73, 0x53, 0xf3, 0x4a, 0x3c, 0x12, 0x8b, 0x33, 0x24, 0x18, 0x15,
	0x18, 0x35, 0x38, 0x83, 0x50, 0xc4, 0x84, 0x64, 0xb8, 0x38, 0xcb, 0x12, 0x73, 0x4a, 0x53, 0x43,
	0x2a, 0x0b, 0x52, 0x25, 0x98, 0xc0, 0x0a, 0x10, 0x02, 0x4e, 0x06, 0x27, 0x1e, 0xc9, 0x31, 0x5e,
	0x78, 0x24, 0xc7, 0xf8, 0xe0, 0x91, 0x1c, 0xe3, 0x84, 0xc7, 0x72, 0x0c, 0x17, 0x1e, 0xcb, 0x31,
	0xdc, 0x78, 0x2c, 0xc7, 0x10, 0x25, 0x06, 0x71, 0x57, 0x05, 0xc2, 0x65, 0x25, 0x95, 0x05, 0xa9,
	0xc5, 0x49, 0x6c, 0x60, 0x67, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x00, 0xdf, 0x79, 0x58,
	0xb8, 0x00, 0x00, 0x00,
}

func (m *DataSpec) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DataSpec) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DataSpec) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ValueType) > 0 {
		i -= len(m.ValueType)
		copy(dAtA[i:], m.ValueType)
		i = encodeVarintDataSpec(dAtA, i, uint64(len(m.ValueType)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.DocumentHash) > 0 {
		i -= len(m.DocumentHash)
		copy(dAtA[i:], m.DocumentHash)
		i = encodeVarintDataSpec(dAtA, i, uint64(len(m.DocumentHash)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintDataSpec(dAtA []byte, offset int, v uint64) int {
	offset -= sovDataSpec(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *DataSpec) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.DocumentHash)
	if l > 0 {
		n += 1 + l + sovDataSpec(uint64(l))
	}
	l = len(m.ValueType)
	if l > 0 {
		n += 1 + l + sovDataSpec(uint64(l))
	}
	return n
}

func sovDataSpec(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDataSpec(x uint64) (n int) {
	return sovDataSpec(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DataSpec) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDataSpec
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
			return fmt.Errorf("proto: DataSpec: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DataSpec: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DocumentHash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataSpec
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
				return ErrInvalidLengthDataSpec
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDataSpec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DocumentHash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValueType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataSpec
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
				return ErrInvalidLengthDataSpec
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDataSpec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ValueType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDataSpec(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDataSpec
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
func skipDataSpec(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDataSpec
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
					return 0, ErrIntOverflowDataSpec
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
					return 0, ErrIntOverflowDataSpec
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
				return 0, ErrInvalidLengthDataSpec
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDataSpec
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDataSpec
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDataSpec        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDataSpec          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDataSpec = fmt.Errorf("proto: unexpected end of group")
)
