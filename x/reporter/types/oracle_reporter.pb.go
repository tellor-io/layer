// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/reporter/oracle_reporter.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
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

// OracleReporter is the struct that holds the data for a reporter
type OracleReporter struct {
	// min_tokens_required to select this reporter
	MinTokensRequired cosmossdk_io_math.Int `protobuf:"bytes,1,opt,name=min_tokens_required,json=minTokensRequired,proto3,customtype=cosmossdk.io/math.Int" json:"min_tokens_required"`
	// commission for the reporter
	CommissionRate cosmossdk_io_math.Uint `protobuf:"bytes,2,opt,name=commission_rate,json=commissionRate,proto3,customtype=cosmossdk.io/math.Uint" json:"commission_rate"`
	// jailed is a bool whether the reporter is jailed or not
	Jailed bool `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	// jailed_until is the time the reporter is jailed until
	JailedUntil time.Time `protobuf:"bytes,4,opt,name=jailed_until,json=jailedUntil,proto3,stdtime" json:"jailed_until"`
}

func (m *OracleReporter) Reset()         { *m = OracleReporter{} }
func (m *OracleReporter) String() string { return proto.CompactTextString(m) }
func (*OracleReporter) ProtoMessage()    {}
func (*OracleReporter) Descriptor() ([]byte, []int) {
	return fileDescriptor_28310cb3dcf79802, []int{0}
}
func (m *OracleReporter) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *OracleReporter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_OracleReporter.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *OracleReporter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OracleReporter.Merge(m, src)
}
func (m *OracleReporter) XXX_Size() int {
	return m.Size()
}
func (m *OracleReporter) XXX_DiscardUnknown() {
	xxx_messageInfo_OracleReporter.DiscardUnknown(m)
}

var xxx_messageInfo_OracleReporter proto.InternalMessageInfo

func (m *OracleReporter) GetJailed() bool {
	if m != nil {
		return m.Jailed
	}
	return false
}

func (m *OracleReporter) GetJailedUntil() time.Time {
	if m != nil {
		return m.JailedUntil
	}
	return time.Time{}
}

type BigUint struct {
	Value cosmossdk_io_math.Uint `protobuf:"bytes,2,opt,name=value,proto3,customtype=cosmossdk.io/math.Uint" json:"value"`
}

func (m *BigUint) Reset()         { *m = BigUint{} }
func (m *BigUint) String() string { return proto.CompactTextString(m) }
func (*BigUint) ProtoMessage()    {}
func (*BigUint) Descriptor() ([]byte, []int) {
	return fileDescriptor_28310cb3dcf79802, []int{1}
}
func (m *BigUint) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *BigUint) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_BigUint.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *BigUint) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BigUint.Merge(m, src)
}
func (m *BigUint) XXX_Size() int {
	return m.Size()
}
func (m *BigUint) XXX_DiscardUnknown() {
	xxx_messageInfo_BigUint.DiscardUnknown(m)
}

var xxx_messageInfo_BigUint proto.InternalMessageInfo

func init() {
	proto.RegisterType((*OracleReporter)(nil), "layer.reporter.OracleReporter")
	proto.RegisterType((*BigUint)(nil), "layer.reporter.BigUint")
}

func init() {
	proto.RegisterFile("layer/reporter/oracle_reporter.proto", fileDescriptor_28310cb3dcf79802)
}

var fileDescriptor_28310cb3dcf79802 = []byte{
	// 410 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x92, 0xb1, 0x6e, 0x14, 0x31,
	0x10, 0x86, 0xcf, 0x01, 0x02, 0xf8, 0xe0, 0x50, 0x16, 0x88, 0x8e, 0x2b, 0x76, 0x4f, 0x11, 0xc5,
	0x89, 0xe8, 0x6c, 0x09, 0xde, 0x60, 0x11, 0x45, 0x24, 0x04, 0xd2, 0x2a, 0xa1, 0x80, 0x62, 0xe5,
	0xdb, 0x33, 0x1b, 0x13, 0xdb, 0xb3, 0xd8, 0x5e, 0x44, 0xde, 0x22, 0x8f, 0x81, 0x44, 0x43, 0xc1,
	0x43, 0xa4, 0x8c, 0xa8, 0x10, 0x45, 0x40, 0x77, 0x05, 0xaf, 0x81, 0xd6, 0xf6, 0xea, 0x0a, 0xba,
	0x34, 0xab, 0xf9, 0x67, 0x67, 0xbe, 0xf9, 0x3d, 0x1a, 0xfc, 0x58, 0xb2, 0x53, 0x6e, 0xa8, 0xe1,
	0x0d, 0x18, 0xc7, 0x0d, 0x05, 0xc3, 0x2a, 0xc9, 0xcb, 0x5e, 0x93, 0xc6, 0x80, 0x83, 0x64, 0xe4,
	0xab, 0x48, 0x9f, 0x9d, 0xec, 0x30, 0x25, 0x34, 0x50, 0xff, 0x0d, 0x25, 0x93, 0x47, 0x15, 0x58,
	0x05, 0xb6, 0xf4, 0x8a, 0x06, 0x11, 0x7f, 0x3d, 0xa8, 0xa1, 0x86, 0x90, 0xef, 0xa2, 0x98, 0xcd,
	0x6a, 0x80, 0x5a, 0x72, 0xea, 0xd5, 0xa2, 0x7d, 0x4f, 0x9d, 0x50, 0xdc, 0x3a, 0xa6, 0x9a, 0x50,
	0xb0, 0xf7, 0x75, 0x0b, 0x8f, 0x5e, 0x7b, 0x3b, 0x45, 0x9c, 0x9b, 0xbc, 0xc3, 0xf7, 0x95, 0xd0,
	0xa5, 0x83, 0x13, 0xae, 0x6d, 0x69, 0xf8, 0xc7, 0x56, 0x18, 0xbe, 0x1c, 0xa3, 0x29, 0x9a, 0xdd,
	0xce, 0xf7, 0xcf, 0x2f, 0xb3, 0xc1, 0xaf, 0xcb, 0xec, 0x61, 0x18, 0x6e, 0x97, 0x27, 0x44, 0x00,
	0x55, 0xcc, 0x1d, 0x93, 0x03, 0xed, 0x7e, 0x7c, 0x9f, 0xe3, 0xe8, 0xea, 0x40, 0xbb, 0x62, 0x47,
	0x09, 0x7d, 0xe8, 0x31, 0x45, 0xa4, 0x24, 0x6f, 0xf0, 0xbd, 0x0a, 0x94, 0x12, 0xd6, 0x0a, 0xd0,
	0xa5, 0x61, 0x8e, 0x8f, 0xb7, 0x3c, 0x78, 0x1e, 0xc1, 0xbb, 0xff, 0x83, 0x8f, 0x84, 0x27, 0x0f,
	0x23, 0xb9, 0x93, 0xc5, 0x68, 0x43, 0x29, 0x98, 0xe3, 0xc9, 0x2e, 0xde, 0xfe, 0xc0, 0x84, 0xe4,
	0xcb, 0xf1, 0xb5, 0x29, 0x9a, 0xdd, 0x2a, 0xa2, 0x4a, 0x5e, 0xe2, 0x3b, 0x21, 0x2a, 0x5b, 0xed,
	0x84, 0x1c, 0x5f, 0x9f, 0xa2, 0xd9, 0xf0, 0xe9, 0x84, 0x84, 0xbd, 0x90, 0x7e, 0x2f, 0xe4, 0xb0,
	0xdf, 0x4b, 0x7e, 0xb7, 0x33, 0x72, 0xf6, 0x3b, 0x43, 0x5f, 0xfe, 0x7e, 0x7b, 0x82, 0x8a, 0x61,
	0x68, 0x3f, 0xea, 0xba, 0xf7, 0x5e, 0xe1, 0x9b, 0xb9, 0xa8, 0x3b, 0x03, 0xc9, 0x73, 0x7c, 0xe3,
	0x13, 0x93, 0xed, 0x15, 0xed, 0x87, 0xde, 0xfc, 0xc5, 0xf9, 0x2a, 0x45, 0x17, 0xab, 0x14, 0xfd,
	0x59, 0xa5, 0xe8, 0x6c, 0x9d, 0x0e, 0x2e, 0xd6, 0xe9, 0xe0, 0xe7, 0x3a, 0x1d, 0xbc, 0xdd, 0xaf,
	0x85, 0x3b, 0x6e, 0x17, 0xa4, 0x02, 0x45, 0x1d, 0x97, 0x12, 0xcc, 0x5c, 0x00, 0x0d, 0x77, 0xf4,
	0x79, 0x73, 0x49, 0xee, 0xb4, 0xe1, 0x76, 0xb1, 0xed, 0x9f, 0xf1, 0xec, 0x5f, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xad, 0x59, 0xc6, 0xcb, 0x68, 0x02, 0x00, 0x00,
}

func (m *OracleReporter) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *OracleReporter) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *OracleReporter) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.JailedUntil, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.JailedUntil):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintOracleReporter(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x22
	if m.Jailed {
		i--
		if m.Jailed {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x18
	}
	{
		size := m.CommissionRate.Size()
		i -= size
		if _, err := m.CommissionRate.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracleReporter(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size := m.MinTokensRequired.Size()
		i -= size
		if _, err := m.MinTokensRequired.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracleReporter(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *BigUint) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *BigUint) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *BigUint) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Value.Size()
		i -= size
		if _, err := m.Value.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintOracleReporter(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	return len(dAtA) - i, nil
}

func encodeVarintOracleReporter(dAtA []byte, offset int, v uint64) int {
	offset -= sovOracleReporter(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *OracleReporter) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MinTokensRequired.Size()
	n += 1 + l + sovOracleReporter(uint64(l))
	l = m.CommissionRate.Size()
	n += 1 + l + sovOracleReporter(uint64(l))
	if m.Jailed {
		n += 2
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdTime(m.JailedUntil)
	n += 1 + l + sovOracleReporter(uint64(l))
	return n
}

func (m *BigUint) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Value.Size()
	n += 1 + l + sovOracleReporter(uint64(l))
	return n
}

func sovOracleReporter(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozOracleReporter(x uint64) (n int) {
	return sovOracleReporter(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *OracleReporter) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracleReporter
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
			return fmt.Errorf("proto: OracleReporter: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: OracleReporter: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinTokensRequired", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracleReporter
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
				return ErrInvalidLengthOracleReporter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracleReporter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MinTokensRequired.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CommissionRate", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracleReporter
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
				return ErrInvalidLengthOracleReporter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracleReporter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.CommissionRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Jailed", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracleReporter
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Jailed = bool(v != 0)
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field JailedUntil", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracleReporter
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
				return ErrInvalidLengthOracleReporter
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthOracleReporter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&m.JailedUntil, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipOracleReporter(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracleReporter
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
func (m *BigUint) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowOracleReporter
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
			return fmt.Errorf("proto: BigUint: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: BigUint: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowOracleReporter
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
				return ErrInvalidLengthOracleReporter
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthOracleReporter
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Value.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipOracleReporter(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthOracleReporter
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
func skipOracleReporter(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowOracleReporter
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
					return 0, ErrIntOverflowOracleReporter
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
					return 0, ErrIntOverflowOracleReporter
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
				return 0, ErrInvalidLengthOracleReporter
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupOracleReporter
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthOracleReporter
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthOracleReporter        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowOracleReporter          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupOracleReporter = fmt.Errorf("proto: unexpected end of group")
)
