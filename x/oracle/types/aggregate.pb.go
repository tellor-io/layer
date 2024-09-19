// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/oracle/aggregate.proto

package types

import (
	encoding_binary "encoding/binary"
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
	ReporterPower uint64 `protobuf:"varint,4,opt,name=reporter_power,json=reporterPower,proto3" json:"reporter_power,omitempty"`
	// standard_deviation is the standard deviation of the reports that were aggregated
	StandardDeviation string `protobuf:"bytes,5,opt,name=standard_deviation,json=standardDeviation,proto3" json:"standard_deviation,omitempty"`
	// list of reporters that were included in the aggregate
	Reporters []*AggregateReporter `protobuf:"bytes,6,rep,name=reporters,proto3" json:"reporters,omitempty"`
	// flagged is true if the aggregate was flagged by a dispute
	Flagged bool `protobuf:"varint,7,opt,name=flagged,proto3" json:"flagged,omitempty"`
	// nonce is the nonce of the aggregate
	Index uint64 `protobuf:"varint,8,opt,name=index,proto3" json:"index,omitempty"`
	// aggregate_report_index is the index of the aggregate report in the micro reports
	AggregateReportIndex uint64 `protobuf:"varint,9,opt,name=aggregate_report_index,json=aggregateReportIndex,proto3" json:"aggregate_report_index,omitempty"`
	// height of the aggregate report
	Height uint64 `protobuf:"varint,10,opt,name=height,proto3" json:"height,omitempty"`
	// height of the micro report
	MicroHeight uint64 `protobuf:"varint,11,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
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

func (m *Aggregate) GetReporterPower() uint64 {
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

func (m *Aggregate) GetAggregateReportIndex() uint64 {
	if m != nil {
		return m.AggregateReportIndex
	}
	return 0
}

func (m *Aggregate) GetHeight() uint64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *Aggregate) GetMicroHeight() uint64 {
	if m != nil {
		return m.MicroHeight
	}
	return 0
}

// LegacyAggregate is the old aggregate struct, it is used to decode old aggregates
type LegacyAggregate struct {
	// query_id is the id of the query
	QueryId []byte `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	// aggregate_value is the value of the aggregate
	AggregateValue string `protobuf:"bytes,2,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	// aggregate_reporter is the address of the reporter
	AggregateReporter string `protobuf:"bytes,3,opt,name=aggregate_reporter,json=aggregateReporter,proto3" json:"aggregate_reporter,omitempty"`
	// reporter_power is the power of the reporter
	ReporterPower int64 `protobuf:"varint,4,opt,name=reporter_power,json=reporterPower,proto3" json:"reporter_power,omitempty"`
	// standard_deviation is the standard deviation of the reports that were aggregated
	StandardDeviation float64 `protobuf:"fixed64,5,opt,name=standard_deviation,json=standardDeviation,proto3" json:"standard_deviation,omitempty"`
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

func (m *LegacyAggregate) Reset()         { *m = LegacyAggregate{} }
func (m *LegacyAggregate) String() string { return proto.CompactTextString(m) }
func (*LegacyAggregate) ProtoMessage()    {}
func (*LegacyAggregate) Descriptor() ([]byte, []int) {
	return fileDescriptor_788ad347f505f8a6, []int{1}
}
func (m *LegacyAggregate) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LegacyAggregate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LegacyAggregate.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LegacyAggregate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LegacyAggregate.Merge(m, src)
}
func (m *LegacyAggregate) XXX_Size() int {
	return m.Size()
}
func (m *LegacyAggregate) XXX_DiscardUnknown() {
	xxx_messageInfo_LegacyAggregate.DiscardUnknown(m)
}

var xxx_messageInfo_LegacyAggregate proto.InternalMessageInfo

func (m *LegacyAggregate) GetQueryId() []byte {
	if m != nil {
		return m.QueryId
	}
	return nil
}

func (m *LegacyAggregate) GetAggregateValue() string {
	if m != nil {
		return m.AggregateValue
	}
	return ""
}

func (m *LegacyAggregate) GetAggregateReporter() string {
	if m != nil {
		return m.AggregateReporter
	}
	return ""
}

func (m *LegacyAggregate) GetReporterPower() int64 {
	if m != nil {
		return m.ReporterPower
	}
	return 0
}

func (m *LegacyAggregate) GetStandardDeviation() float64 {
	if m != nil {
		return m.StandardDeviation
	}
	return 0
}

func (m *LegacyAggregate) GetReporters() []*AggregateReporter {
	if m != nil {
		return m.Reporters
	}
	return nil
}

func (m *LegacyAggregate) GetFlagged() bool {
	if m != nil {
		return m.Flagged
	}
	return false
}

func (m *LegacyAggregate) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *LegacyAggregate) GetAggregateReportIndex() int64 {
	if m != nil {
		return m.AggregateReportIndex
	}
	return 0
}

func (m *LegacyAggregate) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *LegacyAggregate) GetMicroHeight() int64 {
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
	return fileDescriptor_788ad347f505f8a6, []int{2}
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
	proto.RegisterType((*LegacyAggregate)(nil), "layer.oracle.LegacyAggregate")
	proto.RegisterType((*AvailableTimestamps)(nil), "layer.oracle.AvailableTimestamps")
}

func init() { proto.RegisterFile("layer/oracle/aggregate.proto", fileDescriptor_788ad347f505f8a6) }

var fileDescriptor_788ad347f505f8a6 = []byte{
	// 448 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0xcf, 0x6e, 0x13, 0x31,
	0x10, 0xc6, 0x63, 0xd2, 0xe6, 0x8f, 0x13, 0x8a, 0x30, 0x51, 0x65, 0x22, 0xb4, 0x59, 0x2a, 0x55,
	0x2c, 0x87, 0xee, 0x4a, 0xc0, 0x95, 0x43, 0x4a, 0x0f, 0xf4, 0x86, 0x56, 0x88, 0x03, 0x1c, 0x56,
	0x4e, 0x76, 0xea, 0x58, 0x72, 0xe2, 0xc5, 0xeb, 0x84, 0xe6, 0x2d, 0xfa, 0x30, 0x3c, 0x44, 0x8f,
	0x3d, 0x72, 0x02, 0x94, 0xbc, 0x08, 0x8a, 0x1d, 0x6f, 0xa3, 0xd0, 0x5b, 0xe6, 0xfb, 0x7d, 0x33,
	0xe3, 0xc9, 0x7e, 0xf8, 0x85, 0x64, 0x4b, 0xd0, 0x89, 0xd2, 0x6c, 0x2c, 0x21, 0x61, 0x9c, 0x6b,
	0xe0, 0xcc, 0x40, 0x5c, 0x68, 0x65, 0x14, 0xe9, 0x5a, 0x1a, 0x3b, 0xda, 0xef, 0x71, 0xc5, 0x95,
	0x05, 0xc9, 0xe6, 0x97, 0xf3, 0xf4, 0x07, 0x5c, 0x29, 0x2e, 0x21, 0xb1, 0xd5, 0x68, 0x7e, 0x95,
	0x18, 0x31, 0x85, 0xd2, 0xb0, 0x69, 0xb1, 0x35, 0x9c, 0x3e, 0xbc, 0x22, 0xd3, 0x50, 0x28, 0x6d,
	0x40, 0x3b, 0xdb, 0xc9, 0xcf, 0x3a, 0x6e, 0x0f, 0x3d, 0x24, 0xcf, 0x71, 0xeb, 0xfb, 0x1c, 0xf4,
	0x32, 0x13, 0x39, 0x45, 0x21, 0x8a, 0xba, 0x69, 0xd3, 0xd6, 0x97, 0x39, 0x79, 0x85, 0x9f, 0xdc,
	0x0f, 0x59, 0x30, 0x39, 0x07, 0xfa, 0x28, 0x44, 0x51, 0x3b, 0x3d, 0xaa, 0xe4, 0x2f, 0x1b, 0x95,
	0x9c, 0x61, 0xf2, 0xff, 0x36, 0x5a, 0xb7, 0xde, 0xa7, 0x15, 0x49, 0xb7, 0x80, 0x9c, 0xe2, 0x23,
	0x6f, 0xca, 0x0a, 0xf5, 0x03, 0x34, 0x3d, 0x08, 0x51, 0x74, 0x90, 0x3e, 0xf6, 0xea, 0xa7, 0x8d,
	0xb8, 0x99, 0x5a, 0x1a, 0x36, 0xcb, 0x99, 0xce, 0xb3, 0x1c, 0x16, 0x82, 0x19, 0xa1, 0x66, 0xf4,
	0xd0, 0x4d, 0xf5, 0xe4, 0xc2, 0x03, 0xf2, 0x1e, 0xb7, 0x7d, 0x7f, 0x49, 0x1b, 0x61, 0x3d, 0xea,
	0xbc, 0x19, 0xc4, 0xbb, 0x7f, 0x6b, 0x3c, 0xdc, 0x7f, 0x49, 0x7a, 0xdf, 0x41, 0x28, 0x6e, 0x5e,
	0x49, 0xc6, 0x39, 0xe4, 0xb4, 0x19, 0xa2, 0xa8, 0x95, 0xfa, 0x92, 0xf4, 0xf0, 0xa1, 0x98, 0xe5,
	0x70, 0x4d, 0x5b, 0xf6, 0x95, 0xae, 0x20, 0xef, 0xf0, 0xf1, 0xfe, 0xcd, 0x99, 0xb3, 0xb5, 0xad,
	0xad, 0xb7, 0x77, 0xf7, 0xa5, 0xed, 0x3a, 0xc6, 0x8d, 0x09, 0x08, 0x3e, 0x31, 0x14, 0x5b, 0xd7,
	0xb6, 0x22, 0x2f, 0x71, 0x77, 0x2a, 0xc6, 0x5a, 0x65, 0x5b, 0xda, 0xb1, 0xb4, 0x63, 0xb5, 0x8f,
	0x56, 0x3a, 0xf9, 0x86, 0x9f, 0x0d, 0x17, 0x4c, 0x48, 0x36, 0x92, 0xf0, 0xd9, 0x7f, 0xf9, 0x92,
	0x5c, 0x60, 0x5c, 0xe5, 0xa0, 0xa4, 0xc8, 0xde, 0xdd, 0x8f, 0x5d, 0x54, 0x62, 0x1f, 0x95, 0xb8,
	0x6a, 0x38, 0x6f, 0xdd, 0xfe, 0x1e, 0xd4, 0x6e, 0xfe, 0x0c, 0x50, 0xba, 0xd3, 0x77, 0xfe, 0xe1,
	0x76, 0x15, 0xa0, 0xbb, 0x55, 0x80, 0xfe, 0xae, 0x02, 0x74, 0xb3, 0x0e, 0x6a, 0x77, 0xeb, 0xa0,
	0xf6, 0x6b, 0x1d, 0xd4, 0xbe, 0xbe, 0xe6, 0xc2, 0x4c, 0xe6, 0xa3, 0x78, 0xac, 0xa6, 0x89, 0x01,
	0x29, 0x95, 0x3e, 0x13, 0x2a, 0x71, 0x49, 0xbb, 0xf6, 0x59, 0x33, 0xcb, 0x02, 0xca, 0x51, 0xc3,
	0xae, 0x7b, 0xfb, 0x2f, 0x00, 0x00, 0xff, 0xff, 0x6d, 0x1b, 0xfb, 0x4a, 0xeb, 0x02, 0x00, 0x00,
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

func (m *LegacyAggregate) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LegacyAggregate) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LegacyAggregate) MarshalToSizedBuffer(dAtA []byte) (int, error) {
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
	if m.StandardDeviation != 0 {
		i -= 8
		encoding_binary.LittleEndian.PutUint64(dAtA[i:], uint64(math.Float64bits(float64(m.StandardDeviation))))
		i--
		dAtA[i] = 0x29
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

func (m *LegacyAggregate) Size() (n int) {
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
	if m.StandardDeviation != 0 {
		n += 9
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
				m.ReporterPower |= uint64(b&0x7F) << shift
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
				m.AggregateReportIndex |= uint64(b&0x7F) << shift
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
				m.Height |= uint64(b&0x7F) << shift
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
				m.MicroHeight |= uint64(b&0x7F) << shift
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
func (m *LegacyAggregate) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: LegacyAggregate: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LegacyAggregate: illegal tag %d (wire type %d)", fieldNum, wire)
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
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field StandardDeviation", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint64(encoding_binary.LittleEndian.Uint64(dAtA[iNdEx:]))
			iNdEx += 8
			m.StandardDeviation = float64(math.Float64frombits(v))
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
