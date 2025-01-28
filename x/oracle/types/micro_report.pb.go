// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/oracle/micro_report.proto

package types

import (
	fmt "fmt"
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

// MicroReport represents data for a single report
type MicroReport struct {
	// reporter is the address of the reporter
	Reporter string `protobuf:"bytes,1,opt,name=reporter,proto3" json:"reporter,omitempty"`
	// the power of the reporter based on total tokens normalized
	Power uint64 `protobuf:"varint,2,opt,name=power,proto3" json:"power,omitempty"`
	// string identifier of the data spec
	QueryType string `protobuf:"bytes,3,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
	// hash of the query data
	QueryId []byte `protobuf:"bytes,4,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	// aggregate method to use for aggregating all the reports for the query id
	AggregateMethod string `protobuf:"bytes,5,opt,name=aggregate_method,json=aggregateMethod,proto3" json:"aggregate_method,omitempty"`
	// hex string of the response value
	Value string `protobuf:"bytes,6,opt,name=value,proto3" json:"value,omitempty"`
	// timestamp of when the report was created
	Timestamp time.Time `protobuf:"bytes,7,opt,name=timestamp,proto3,stdtime" json:"timestamp"`
	// indicates if the report's query id is in the cyclelist
	Cyclelist bool `protobuf:"varint,8,opt,name=cyclelist,proto3" json:"cyclelist,omitempty"`
	// block number of when the report was created
	BlockNumber uint64 `protobuf:"varint,9,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	// identifier for the report's aggregate group
	MetaId uint64 `protobuf:"varint,10,opt,name=meta_id,json=metaId,proto3" json:"meta_id,omitempty"`
}

func (m *MicroReport) Reset()         { *m = MicroReport{} }
func (m *MicroReport) String() string { return proto.CompactTextString(m) }
func (*MicroReport) ProtoMessage()    {}
func (*MicroReport) Descriptor() ([]byte, []int) {
	return fileDescriptor_c39350954f878191, []int{0}
}
func (m *MicroReport) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MicroReport) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MicroReport.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MicroReport) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MicroReport.Merge(m, src)
}
func (m *MicroReport) XXX_Size() int {
	return m.Size()
}
func (m *MicroReport) XXX_DiscardUnknown() {
	xxx_messageInfo_MicroReport.DiscardUnknown(m)
}

var xxx_messageInfo_MicroReport proto.InternalMessageInfo

func (m *MicroReport) GetReporter() string {
	if m != nil {
		return m.Reporter
	}
	return ""
}

func (m *MicroReport) GetPower() uint64 {
	if m != nil {
		return m.Power
	}
	return 0
}

func (m *MicroReport) GetQueryType() string {
	if m != nil {
		return m.QueryType
	}
	return ""
}

func (m *MicroReport) GetQueryId() []byte {
	if m != nil {
		return m.QueryId
	}
	return nil
}

func (m *MicroReport) GetAggregateMethod() string {
	if m != nil {
		return m.AggregateMethod
	}
	return ""
}

func (m *MicroReport) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *MicroReport) GetTimestamp() time.Time {
	if m != nil {
		return m.Timestamp
	}
	return time.Time{}
}

func (m *MicroReport) GetCyclelist() bool {
	if m != nil {
		return m.Cyclelist
	}
	return false
}

func (m *MicroReport) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

func (m *MicroReport) GetMetaId() uint64 {
	if m != nil {
		return m.MetaId
	}
	return 0
}

func init() {
	proto.RegisterType((*MicroReport)(nil), "layer.oracle.MicroReport")
}

func init() { proto.RegisterFile("layer/oracle/micro_report.proto", fileDescriptor_c39350954f878191) }

var fileDescriptor_c39350954f878191 = []byte{
	// 373 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x44, 0x51, 0xbd, 0xae, 0xda, 0x30,
	0x14, 0x8e, 0x29, 0x3f, 0x89, 0x41, 0x6a, 0x65, 0x21, 0xd5, 0x8d, 0xda, 0x24, 0xed, 0x14, 0x86,
	0x26, 0x52, 0xfb, 0x06, 0x74, 0x62, 0xa0, 0x43, 0xc4, 0xd4, 0x25, 0xca, 0xcf, 0xa9, 0x89, 0xea,
	0xd4, 0xa9, 0x71, 0x7a, 0x6f, 0xde, 0x82, 0xc7, 0x62, 0x64, 0x64, 0xba, 0xf7, 0x0a, 0x5e, 0xe4,
	0x2a, 0xb6, 0x80, 0xcd, 0xdf, 0xdf, 0x39, 0xfa, 0x7c, 0xb0, 0xcf, 0xb3, 0x0e, 0x64, 0x2c, 0x64,
	0x56, 0x70, 0x88, 0xeb, 0xaa, 0x90, 0x22, 0x95, 0xd0, 0x08, 0xa9, 0xa2, 0x46, 0x0a, 0x25, 0xc8,
	0x4c, 0x1b, 0x22, 0x63, 0x70, 0xe7, 0x4c, 0x30, 0xa1, 0x85, 0xb8, 0x7f, 0x19, 0x8f, 0xeb, 0x33,
	0x21, 0x18, 0x87, 0x58, 0xa3, 0xbc, 0xfd, 0x1d, 0xab, 0xaa, 0x86, 0x9d, 0xca, 0xea, 0xc6, 0x18,
	0xbe, 0x9c, 0x06, 0x78, 0xba, 0xee, 0x67, 0x27, 0x7a, 0x34, 0x71, 0xb1, 0x6d, 0x96, 0x80, 0xa4,
	0x28, 0x40, 0xa1, 0x93, 0xdc, 0x30, 0x99, 0xe3, 0x51, 0x23, 0x1e, 0x40, 0xd2, 0x41, 0x80, 0xc2,
	0x61, 0x62, 0x00, 0xf9, 0x84, 0xf1, 0xbf, 0x16, 0x64, 0x97, 0xaa, 0xae, 0x01, 0xfa, 0x46, 0x67,
	0x1c, 0xcd, 0x6c, 0xba, 0x06, 0xc8, 0x07, 0x6c, 0x1b, 0xb9, 0x2a, 0xe9, 0x30, 0x40, 0xe1, 0x2c,
	0x99, 0x68, 0xbc, 0x2a, 0xc9, 0x02, 0xbf, 0xcb, 0x18, 0x93, 0xc0, 0x32, 0x05, 0x69, 0x0d, 0x6a,
	0x2b, 0x4a, 0x3a, 0xd2, 0xf9, 0xb7, 0x37, 0x7e, 0xad, 0xe9, 0x7e, 0xf5, 0xff, 0x8c, 0xb7, 0x40,
	0xc7, 0x5a, 0x37, 0x80, 0x2c, 0xb1, 0x73, 0xeb, 0x43, 0x27, 0x01, 0x0a, 0xa7, 0xdf, 0xdc, 0xc8,
	0x34, 0x8e, 0xae, 0x8d, 0xa3, 0xcd, 0xd5, 0xb1, 0xb4, 0x0f, 0x4f, 0xbe, 0xb5, 0x7f, 0xf6, 0x51,
	0x72, 0x8f, 0x91, 0x8f, 0xd8, 0x29, 0xba, 0x82, 0x03, 0xaf, 0x76, 0x8a, 0xda, 0x01, 0x0a, 0xed,
	0xe4, 0x4e, 0x90, 0xcf, 0x78, 0x96, 0x73, 0x51, 0xfc, 0x49, 0xff, 0xb6, 0x75, 0x0e, 0x92, 0x3a,
	0xba, 0xf9, 0x54, 0x73, 0x3f, 0x35, 0x45, 0xde, 0xe3, 0x49, 0x0d, 0x2a, 0xeb, 0xfb, 0x61, 0xad,
	0x8e, 0x7b, 0xb8, 0x2a, 0x97, 0x3f, 0x0e, 0x67, 0x0f, 0x1d, 0xcf, 0x1e, 0x7a, 0x39, 0x7b, 0x68,
	0x7f, 0xf1, 0xac, 0xe3, 0xc5, 0xb3, 0x4e, 0x17, 0xcf, 0xfa, 0xb5, 0x60, 0x95, 0xda, 0xb6, 0x79,
	0x54, 0x88, 0x3a, 0x56, 0xc0, 0xb9, 0x90, 0x5f, 0x2b, 0x11, 0x9b, 0x7b, 0x3f, 0x5e, 0x2f, 0xde,
	0x7f, 0xe7, 0x2e, 0x1f, 0xeb, 0x1e, 0xdf, 0x5f, 0x03, 0x00, 0x00, 0xff, 0xff, 0x7d, 0xd5, 0x14,
	0x8b, 0x0e, 0x02, 0x00, 0x00,
}

func (m *MicroReport) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MicroReport) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MicroReport) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.MetaId != 0 {
		i = encodeVarintMicroReport(dAtA, i, uint64(m.MetaId))
		i--
		dAtA[i] = 0x50
	}
	if m.BlockNumber != 0 {
		i = encodeVarintMicroReport(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x48
	}
	if m.Cyclelist {
		i--
		if m.Cyclelist {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x40
	}
	n1, err1 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.Timestamp, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.Timestamp):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintMicroReport(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x3a
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintMicroReport(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.AggregateMethod) > 0 {
		i -= len(m.AggregateMethod)
		copy(dAtA[i:], m.AggregateMethod)
		i = encodeVarintMicroReport(dAtA, i, uint64(len(m.AggregateMethod)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.QueryId) > 0 {
		i -= len(m.QueryId)
		copy(dAtA[i:], m.QueryId)
		i = encodeVarintMicroReport(dAtA, i, uint64(len(m.QueryId)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.QueryType) > 0 {
		i -= len(m.QueryType)
		copy(dAtA[i:], m.QueryType)
		i = encodeVarintMicroReport(dAtA, i, uint64(len(m.QueryType)))
		i--
		dAtA[i] = 0x1a
	}
	if m.Power != 0 {
		i = encodeVarintMicroReport(dAtA, i, uint64(m.Power))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Reporter) > 0 {
		i -= len(m.Reporter)
		copy(dAtA[i:], m.Reporter)
		i = encodeVarintMicroReport(dAtA, i, uint64(len(m.Reporter)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintMicroReport(dAtA []byte, offset int, v uint64) int {
	offset -= sovMicroReport(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MicroReport) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Reporter)
	if l > 0 {
		n += 1 + l + sovMicroReport(uint64(l))
	}
	if m.Power != 0 {
		n += 1 + sovMicroReport(uint64(m.Power))
	}
	l = len(m.QueryType)
	if l > 0 {
		n += 1 + l + sovMicroReport(uint64(l))
	}
	l = len(m.QueryId)
	if l > 0 {
		n += 1 + l + sovMicroReport(uint64(l))
	}
	l = len(m.AggregateMethod)
	if l > 0 {
		n += 1 + l + sovMicroReport(uint64(l))
	}
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovMicroReport(uint64(l))
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdTime(m.Timestamp)
	n += 1 + l + sovMicroReport(uint64(l))
	if m.Cyclelist {
		n += 2
	}
	if m.BlockNumber != 0 {
		n += 1 + sovMicroReport(uint64(m.BlockNumber))
	}
	if m.MetaId != 0 {
		n += 1 + sovMicroReport(uint64(m.MetaId))
	}
	return n
}

func sovMicroReport(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMicroReport(x uint64) (n int) {
	return sovMicroReport(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MicroReport) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMicroReport
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
			return fmt.Errorf("proto: MicroReport: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MicroReport: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reporter", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
				return ErrInvalidLengthMicroReport
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMicroReport
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
					return ErrIntOverflowMicroReport
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Power |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
				return ErrInvalidLengthMicroReport
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMicroReport
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryId", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
				return ErrInvalidLengthMicroReport
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMicroReport
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryId = append(m.QueryId[:0], dAtA[iNdEx:postIndex]...)
			if m.QueryId == nil {
				m.QueryId = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AggregateMethod", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
				return ErrInvalidLengthMicroReport
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMicroReport
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AggregateMethod = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
				return ErrInvalidLengthMicroReport
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMicroReport
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
				return ErrInvalidLengthMicroReport
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMicroReport
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&m.Timestamp, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Cyclelist", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
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
			m.Cyclelist = bool(v != 0)
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockNumber", wireType)
			}
			m.BlockNumber = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BlockNumber |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MetaId", wireType)
			}
			m.MetaId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMicroReport
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MetaId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMicroReport(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMicroReport
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
func skipMicroReport(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMicroReport
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
					return 0, ErrIntOverflowMicroReport
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
					return 0, ErrIntOverflowMicroReport
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
				return 0, ErrInvalidLengthMicroReport
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMicroReport
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMicroReport
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMicroReport        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMicroReport          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMicroReport = fmt.Errorf("proto: unexpected end of group")
)
