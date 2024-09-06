// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/oracle/aggregate.proto

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

// aggregate struct to represent meta data of an aggregate report
type Aggregate struct {
	// query_id is the id of the query
	QueryId []byte `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	// aggregate_value is the value of the aggregate
	AggregateValue string `protobuf:"bytes,2,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	// aggregate_reporter is the address of the reporter
	AggregateReporter string `protobuf:"bytes,3,opt,name=aggregate_reporter,json=aggregateReporter,proto3" json:"aggregate_reporter,omitempty"`
	// reporter_power is the power of the reporter
	ReporterPower int64 `protobuf:"varint,4,opt,name=reporter_power,json=reporterPower,proto3" json:"reporter_power,omitempty"`
	// standard_deviation is the standard deviation of the reports that were aggregated
	StandardDeviation string `protobuf:"bytes,5,opt,name=standard_deviation,json=standardDeviation,proto3" json:"standard_deviation,omitempty"`
	// list of reporters that were included in the aggregate
	Reporters []*AggregateReporter `protobuf:"bytes,6,rep,name=reporters,proto3" json:"reporters,omitempty"`
	// flagged is true if the aggregate was flagged by a dispute
	Flagged bool `protobuf:"varint,7,opt,name=flagged,proto3" json:"flagged,omitempty"`
	// nonce is the nonce of the aggregate
	Index uint64 `protobuf:"varint,8,opt,name=index,proto3" json:"index,omitempty"`
	// aggregate_report_index is the index of the aggregate report in the micro reports
	AggregateReportIndex int64 `protobuf:"varint,9,opt,name=aggregate_report_index,json=aggregateReportIndex,proto3" json:"aggregate_report_index,omitempty"`
	// height of the aggregate report
	Height int64 `protobuf:"varint,10,opt,name=height,proto3" json:"height,omitempty"`
	// height of the micro report
	MicroHeight int64 `protobuf:"varint,11,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
}

func (m *Aggregate) Reset()         { *m = Aggregate{} }
func (m *Aggregate) String() string { return proto.CompactTextString(m) }
func (*Aggregate) ProtoMessage()    {}
func (*Aggregate) Descriptor() ([]byte, []int) {
	return fileDescriptor_788ad347f505f8a6, []int{0}
}
func (m *Aggregate) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Aggregate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Aggregate.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Aggregate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Aggregate.Merge(m, src)
}
func (m *Aggregate) XXX_Size() int {
	return m.Size()
}
func (m *Aggregate) XXX_DiscardUnknown() {
	xxx_messageInfo_Aggregate.DiscardUnknown(m)
}

var xxx_messageInfo_Aggregate proto.InternalMessageInfo

func (m *Aggregate) GetQueryId() []byte {
	if m != nil {
		return m.QueryId
	}
	return nil
}

func (m *Aggregate) GetAggregateValue() string {
	if m != nil {
		return m.AggregateValue
	}
	return ""
}

func (m *Aggregate) GetAggregateReporter() string {
	if m != nil {
		return m.AggregateReporter
	}
	return ""
}

func (m *Aggregate) GetReporterPower() int64 {
	if m != nil {
		return m.ReporterPower
	}
	return 0
}

func (m *Aggregate) GetStandardDeviation() string {
	if m != nil {
		return m.StandardDeviation
	}
	return ""
}

func (m *Aggregate) GetReporters() []*AggregateReporter {
	if m != nil {
		return m.Reporters
	}
	return nil
}

func (m *Aggregate) GetFlagged() bool {
	if m != nil {
		return m.Flagged
	}
	return false
}

func (m *Aggregate) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *Aggregate) GetAggregateReportIndex() int64 {
	if m != nil {
		return m.AggregateReportIndex
	}
	return 0
}

func (m *Aggregate) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *Aggregate) GetMicroHeight() int64 {
	if m != nil {
		return m.MicroHeight
	}
	return 0
}

type AvailableTimestamps struct {
	Timestamps []time.Time `protobuf:"bytes,1,rep,name=timestamps,proto3,stdtime" json:"timestamps"`
}

func (m *AvailableTimestamps) Reset()         { *m = AvailableTimestamps{} }
func (m *AvailableTimestamps) String() string { return proto.CompactTextString(m) }
func (*AvailableTimestamps) ProtoMessage()    {}
func (*AvailableTimestamps) Descriptor() ([]byte, []int) {
	return fileDescriptor_788ad347f505f8a6, []int{1}
}
func (m *AvailableTimestamps) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AvailableTimestamps) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AvailableTimestamps.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AvailableTimestamps) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AvailableTimestamps.Merge(m, src)
}
func (m *AvailableTimestamps) XXX_Size() int {
	return m.Size()
}
func (m *AvailableTimestamps) XXX_DiscardUnknown() {
	xxx_messageInfo_AvailableTimestamps.DiscardUnknown(m)
}

var xxx_messageInfo_AvailableTimestamps proto.InternalMessageInfo

func (m *AvailableTimestamps) GetTimestamps() []time.Time {
	if m != nil {
		return m.Timestamps
	}
	return nil
}

func init() {
	proto.RegisterType((*Aggregate)(nil), "layer.oracle.Aggregate")
	proto.RegisterType((*AvailableTimestamps)(nil), "layer.oracle.AvailableTimestamps")
}

func init() { proto.RegisterFile("layer/oracle/aggregate.proto", fileDescriptor_788ad347f505f8a6) }

var fileDescriptor_788ad347f505f8a6 = []byte{
	// 451 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0xcf, 0x6e, 0xd3, 0x40,
	0x10, 0xc6, 0xb3, 0xa4, 0xcd, 0x9f, 0x4d, 0x28, 0x62, 0x89, 0xaa, 0x25, 0x42, 0x8e, 0xa9, 0x54,
	0x61, 0x0e, 0xb5, 0x25, 0xe0, 0xca, 0x21, 0xa5, 0x07, 0x7a, 0x43, 0x16, 0xe2, 0x00, 0x07, 0x6b,
	0x13, 0x4f, 0x37, 0x2b, 0x6d, 0xb2, 0x66, 0xbd, 0x09, 0xcd, 0x5b, 0xf4, 0x61, 0x78, 0x88, 0x1e,
	0x7b, 0xe4, 0x04, 0x28, 0x79, 0x11, 0x94, 0xdd, 0xac, 0x1b, 0x85, 0xde, 0x3c, 0xdf, 0xef, 0x9b,
	0x19, 0x8f, 0xfd, 0xe1, 0x17, 0x92, 0x2d, 0x41, 0x27, 0x4a, 0xb3, 0xb1, 0x84, 0x84, 0x71, 0xae,
	0x81, 0x33, 0x03, 0x71, 0xa1, 0x95, 0x51, 0xa4, 0x6b, 0x69, 0xec, 0x68, 0xbf, 0xc7, 0x15, 0x57,
	0x16, 0x24, 0x9b, 0x27, 0xe7, 0xe9, 0x0f, 0xb8, 0x52, 0x5c, 0x42, 0x62, 0xab, 0xd1, 0xfc, 0x2a,
	0x31, 0x62, 0x0a, 0xa5, 0x61, 0xd3, 0x62, 0x6b, 0x38, 0x7d, 0x78, 0x45, 0xa6, 0xa1, 0x50, 0xda,
	0x80, 0x76, 0xb6, 0x93, 0x9f, 0x75, 0xdc, 0x1e, 0x7a, 0x48, 0x9e, 0xe3, 0xd6, 0xf7, 0x39, 0xe8,
	0x65, 0x26, 0x72, 0x8a, 0x42, 0x14, 0x75, 0xd3, 0xa6, 0xad, 0x2f, 0x73, 0xf2, 0x0a, 0x3f, 0xb9,
	0x1f, 0xb2, 0x60, 0x72, 0x0e, 0xf4, 0x51, 0x88, 0xa2, 0x76, 0x7a, 0x54, 0xc9, 0x5f, 0x36, 0x2a,
	0x39, 0xc3, 0xe4, 0xff, 0x6d, 0xb4, 0x6e, 0xbd, 0x4f, 0x2b, 0x92, 0x6e, 0x01, 0x39, 0xc5, 0x47,
	0xde, 0x94, 0x15, 0xea, 0x07, 0x68, 0x7a, 0x10, 0xa2, 0xa8, 0x9e, 0x3e, 0xf6, 0xea, 0xa7, 0x8d,
	0xb8, 0x99, 0x5a, 0x1a, 0x36, 0xcb, 0x99, 0xce, 0xb3, 0x1c, 0x16, 0x82, 0x19, 0xa1, 0x66, 0xf4,
	0xd0, 0x4d, 0xf5, 0xe4, 0xc2, 0x03, 0xf2, 0x1e, 0xb7, 0x7d, 0x7f, 0x49, 0x1b, 0x61, 0x3d, 0xea,
	0xbc, 0x19, 0xc4, 0xbb, 0x9f, 0x35, 0x1e, 0xee, 0xbf, 0x49, 0x7a, 0xdf, 0x41, 0x28, 0x6e, 0x5e,
	0x49, 0xc6, 0x39, 0xe4, 0xb4, 0x19, 0xa2, 0xa8, 0x95, 0xfa, 0x92, 0xf4, 0xf0, 0xa1, 0x98, 0xe5,
	0x70, 0x4d, 0x5b, 0x21, 0x8a, 0x0e, 0x52, 0x57, 0x90, 0x77, 0xf8, 0x78, 0xff, 0xe6, 0xcc, 0xd9,
	0xda, 0xf6, 0x98, 0xde, 0xde, 0xdd, 0x97, 0xb6, 0xeb, 0x18, 0x37, 0x26, 0x20, 0xf8, 0xc4, 0x50,
	0x6c, 0x5d, 0xdb, 0x8a, 0xbc, 0xc4, 0xdd, 0xa9, 0x18, 0x6b, 0x95, 0x6d, 0x69, 0xc7, 0xd2, 0x8e,
	0xd5, 0x3e, 0x5a, 0xe9, 0xe4, 0x1b, 0x7e, 0x36, 0x5c, 0x30, 0x21, 0xd9, 0x48, 0xc2, 0x67, 0xff,
	0xe7, 0x4b, 0x72, 0x81, 0x71, 0x95, 0x83, 0x92, 0x22, 0x7b, 0x77, 0x3f, 0x76, 0x51, 0x89, 0x7d,
	0x54, 0xe2, 0xaa, 0xe1, 0xbc, 0x75, 0xfb, 0x7b, 0x50, 0xbb, 0xf9, 0x33, 0x40, 0xe9, 0x4e, 0xdf,
	0xf9, 0x87, 0xdb, 0x55, 0x80, 0xee, 0x56, 0x01, 0xfa, 0xbb, 0x0a, 0xd0, 0xcd, 0x3a, 0xa8, 0xdd,
	0xad, 0x83, 0xda, 0xaf, 0x75, 0x50, 0xfb, 0xfa, 0x9a, 0x0b, 0x33, 0x99, 0x8f, 0xe2, 0xb1, 0x9a,
	0x26, 0x06, 0xa4, 0x54, 0xfa, 0x4c, 0xa8, 0xc4, 0x25, 0xed, 0xda, 0x67, 0xcd, 0x2c, 0x0b, 0x28,
	0x47, 0x0d, 0xbb, 0xee, 0xed, 0xbf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xcd, 0x8f, 0x0c, 0x7a, 0xeb,
	0x02, 0x00, 0x00,
}

func (m *Aggregate) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Aggregate) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Aggregate) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.MicroHeight != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.MicroHeight))
		i--
		dAtA[i] = 0x58
	}
	if m.Height != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x50
	}
	if m.AggregateReportIndex != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.AggregateReportIndex))
		i--
		dAtA[i] = 0x48
	}
	if m.Index != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x40
	}
	if m.Flagged {
		i--
		if m.Flagged {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x38
	}
	if len(m.Reporters) > 0 {
		for iNdEx := len(m.Reporters) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Reporters[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintAggregate(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x32
		}
	}
	if len(m.StandardDeviation) > 0 {
		i -= len(m.StandardDeviation)
		copy(dAtA[i:], m.StandardDeviation)
		i = encodeVarintAggregate(dAtA, i, uint64(len(m.StandardDeviation)))
		i--
		dAtA[i] = 0x2a
	}
	if m.ReporterPower != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.ReporterPower))
		i--
		dAtA[i] = 0x20
	}
	if len(m.AggregateReporter) > 0 {
		i -= len(m.AggregateReporter)
		copy(dAtA[i:], m.AggregateReporter)
		i = encodeVarintAggregate(dAtA, i, uint64(len(m.AggregateReporter)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.AggregateValue) > 0 {
		i -= len(m.AggregateValue)
		copy(dAtA[i:], m.AggregateValue)
		i = encodeVarintAggregate(dAtA, i, uint64(len(m.AggregateValue)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.QueryId) > 0 {
		i -= len(m.QueryId)
		copy(dAtA[i:], m.QueryId)
		i = encodeVarintAggregate(dAtA, i, uint64(len(m.QueryId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *AvailableTimestamps) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AvailableTimestamps) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AvailableTimestamps) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Timestamps) > 0 {
		for iNdEx := len(m.Timestamps) - 1; iNdEx >= 0; iNdEx-- {
			n, err := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.Timestamps[iNdEx], dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.Timestamps[iNdEx]):])
			if err != nil {
				return 0, err
			}
			i -= n
			i = encodeVarintAggregate(dAtA, i, uint64(n))
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintAggregate(dAtA []byte, offset int, v uint64) int {
	offset -= sovAggregate(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Aggregate) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.QueryId)
	if l > 0 {
		n += 1 + l + sovAggregate(uint64(l))
	}
	l = len(m.AggregateValue)
	if l > 0 {
		n += 1 + l + sovAggregate(uint64(l))
	}
	l = len(m.AggregateReporter)
	if l > 0 {
		n += 1 + l + sovAggregate(uint64(l))
	}
	if m.ReporterPower != 0 {
		n += 1 + sovAggregate(uint64(m.ReporterPower))
	}
	l = len(m.StandardDeviation)
	if l > 0 {
		n += 1 + l + sovAggregate(uint64(l))
	}
	if len(m.Reporters) > 0 {
		for _, e := range m.Reporters {
			l = e.Size()
			n += 1 + l + sovAggregate(uint64(l))
		}
	}
	if m.Flagged {
		n += 2
	}
	if m.Index != 0 {
		n += 1 + sovAggregate(uint64(m.Index))
	}
	if m.AggregateReportIndex != 0 {
		n += 1 + sovAggregate(uint64(m.AggregateReportIndex))
	}
	if m.Height != 0 {
		n += 1 + sovAggregate(uint64(m.Height))
	}
	if m.MicroHeight != 0 {
		n += 1 + sovAggregate(uint64(m.MicroHeight))
	}
	return n
}

func (m *AvailableTimestamps) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Timestamps) > 0 {
		for _, e := range m.Timestamps {
			l = github_com_cosmos_gogoproto_types.SizeOfStdTime(e)
			n += 1 + l + sovAggregate(uint64(l))
		}
	}
	return n
}

func sovAggregate(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAggregate(x uint64) (n int) {
	return sovAggregate(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Aggregate) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAggregate
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
			return fmt.Errorf("proto: Aggregate: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Aggregate: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryId", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
				return ErrInvalidLengthAggregate
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthAggregate
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryId = append(m.QueryId[:0], dAtA[iNdEx:postIndex]...)
			if m.QueryId == nil {
				m.QueryId = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AggregateValue", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
				return ErrInvalidLengthAggregate
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAggregate
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AggregateValue = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AggregateReporter", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
				return ErrInvalidLengthAggregate
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAggregate
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AggregateReporter = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReporterPower", wireType)
			}
			m.ReporterPower = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ReporterPower |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field StandardDeviation", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
				return ErrInvalidLengthAggregate
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthAggregate
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.StandardDeviation = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reporters", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
				return ErrInvalidLengthAggregate
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAggregate
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Reporters = append(m.Reporters, &AggregateReporter{})
			if err := m.Reporters[len(m.Reporters)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Flagged", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
			m.Flagged = bool(v != 0)
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Index |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AggregateReportIndex", wireType)
			}
			m.AggregateReportIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.AggregateReportIndex |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MicroHeight", wireType)
			}
			m.MicroHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MicroHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipAggregate(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAggregate
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
func (m *AvailableTimestamps) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAggregate
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
			return fmt.Errorf("proto: AvailableTimestamps: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AvailableTimestamps: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamps", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
				return ErrInvalidLengthAggregate
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAggregate
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Timestamps = append(m.Timestamps, time.Time{})
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&(m.Timestamps[len(m.Timestamps)-1]), dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAggregate(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAggregate
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
func skipAggregate(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAggregate
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
					return 0, ErrIntOverflowAggregate
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
					return 0, ErrIntOverflowAggregate
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
				return 0, ErrInvalidLengthAggregate
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAggregate
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAggregate
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAggregate        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAggregate          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAggregate = fmt.Errorf("proto: unexpected end of group")
)
