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
	// aggregate_power is the power of all the reporters
	// that reported for the aggregate
	AggregatePower uint64 `protobuf:"varint,4,opt,name=aggregate_power,json=aggregatePower,proto3" json:"aggregate_power,omitempty"`
	// flagged is true if the aggregate was flagged by a dispute
	Flagged bool `protobuf:"varint,5,opt,name=flagged,proto3" json:"flagged,omitempty"`
	// index is the index of the aggregate
	Index uint64 `protobuf:"varint,6,opt,name=index,proto3" json:"index,omitempty"`
	// height of the aggregate report
	Height uint64 `protobuf:"varint,7,opt,name=height,proto3" json:"height,omitempty"`
	// height of the micro report
	MicroHeight uint64 `protobuf:"varint,8,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
	// meta_id is the id of the querymeta iterator
	MetaId uint64 `protobuf:"varint,9,opt,name=meta_id,json=metaId,proto3" json:"meta_id,omitempty"`
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

func (m *Aggregate) GetAggregatePower() uint64 {
	if m != nil {
		return m.AggregatePower
	}
	return 0
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

func (m *Aggregate) GetMetaId() uint64 {
	if m != nil {
		return m.MetaId
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
	// 396 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0xcd, 0x6e, 0xd4, 0x30,
	0x14, 0x85, 0xe3, 0xfe, 0x4c, 0x32, 0xee, 0x08, 0x84, 0xa9, 0xc0, 0x8c, 0x50, 0x26, 0x54, 0x42,
	0x84, 0x45, 0x13, 0x09, 0x9e, 0xa0, 0x85, 0x05, 0xdd, 0xa1, 0x08, 0xb1, 0x80, 0xc5, 0xc8, 0x99,
	0xdc, 0x7a, 0x2c, 0x39, 0x38, 0x38, 0x4e, 0xe9, 0xbc, 0x45, 0xdf, 0x83, 0x17, 0xe9, 0x72, 0x96,
	0xac, 0x00, 0xcd, 0xbc, 0x08, 0x8a, 0x9d, 0x84, 0x41, 0x74, 0xe7, 0x73, 0xbe, 0x73, 0x7d, 0xa5,
	0x7b, 0xf0, 0x53, 0xc9, 0x56, 0xa0, 0x53, 0xa5, 0xd9, 0x42, 0x42, 0xca, 0x38, 0xd7, 0xc0, 0x99,
	0x81, 0xa4, 0xd2, 0xca, 0x28, 0x32, 0xb1, 0x34, 0x71, 0x74, 0x7a, 0xcc, 0x15, 0x57, 0x16, 0xa4,
	0xed, 0xcb, 0x65, 0xa6, 0x33, 0xae, 0x14, 0x97, 0x90, 0x5a, 0x95, 0x37, 0x97, 0xa9, 0x11, 0x25,
	0xd4, 0x86, 0x95, 0x55, 0x17, 0x78, 0x7e, 0xf7, 0x8a, 0xb9, 0x86, 0x4a, 0x69, 0x03, 0xda, 0xc5,
	0x4e, 0xbe, 0xef, 0xe1, 0xf1, 0x59, 0x0f, 0xc9, 0x13, 0x1c, 0x7c, 0x6d, 0x40, 0xaf, 0xe6, 0xa2,
	0xa0, 0x28, 0x42, 0xf1, 0x24, 0xf3, 0xad, 0xbe, 0x28, 0xc8, 0x0b, 0x7c, 0xff, 0xef, 0x27, 0x57,
	0x4c, 0x36, 0x40, 0xf7, 0x22, 0x14, 0x8f, 0xb3, 0x7b, 0x83, 0xfd, 0xb1, 0x75, 0xc9, 0x29, 0x26,
	0xff, 0x6f, 0xa3, 0xfb, 0x36, 0xfb, 0x60, 0x20, 0x59, 0x07, 0xfe, 0xfd, 0xb7, 0x52, 0xdf, 0x40,
	0xd3, 0x83, 0x08, 0xc5, 0x07, 0x3b, 0xff, 0xbe, 0x6f, 0x5d, 0x42, 0xb1, 0x7f, 0x29, 0x19, 0xe7,
	0x50, 0xd0, 0xc3, 0x08, 0xc5, 0x41, 0xd6, 0x4b, 0x72, 0x8c, 0x0f, 0xc5, 0x97, 0x02, 0xae, 0xe9,
	0xc8, 0x0e, 0x3a, 0x41, 0x1e, 0xe1, 0xd1, 0x12, 0x04, 0x5f, 0x1a, 0xea, 0x5b, 0xbb, 0x53, 0xe4,
	0x19, 0x9e, 0x94, 0x62, 0xa1, 0xd5, 0xbc, 0xa3, 0x81, 0xa5, 0x47, 0xd6, 0x7b, 0xe7, 0x22, 0x8f,
	0xb1, 0x5f, 0x82, 0x61, 0xed, 0x15, 0xc6, 0x6e, 0xb6, 0x95, 0x17, 0xc5, 0xc9, 0x67, 0xfc, 0xf0,
	0xec, 0x8a, 0x09, 0xc9, 0x72, 0x09, 0x1f, 0xfa, 0x83, 0xd7, 0xe4, 0x2d, 0xc6, 0xc3, 0xf9, 0x6b,
	0x8a, 0xa2, 0xfd, 0xf8, 0xe8, 0xd5, 0x34, 0x71, 0x0d, 0x25, 0x7d, 0x43, 0xc9, 0x30, 0x70, 0x1e,
	0xdc, 0xfe, 0x9c, 0x79, 0x37, 0xbf, 0x66, 0x28, 0xdb, 0x99, 0x3b, 0x7f, 0x73, 0xbb, 0x09, 0xd1,
	0x7a, 0x13, 0xa2, 0xdf, 0x9b, 0x10, 0xdd, 0x6c, 0x43, 0x6f, 0xbd, 0x0d, 0xbd, 0x1f, 0xdb, 0xd0,
	0xfb, 0xf4, 0x92, 0x0b, 0xb3, 0x6c, 0xf2, 0x64, 0xa1, 0xca, 0xd4, 0x80, 0x94, 0x4a, 0x9f, 0x0a,
	0x95, 0xba, 0x82, 0xaf, 0xfb, 0x8a, 0xcd, 0xaa, 0x82, 0x3a, 0x1f, 0xd9, 0x75, 0xaf, 0xff, 0x04,
	0x00, 0x00, 0xff, 0xff, 0x06, 0xc1, 0x6f, 0x71, 0x62, 0x02, 0x00, 0x00,
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
	if m.MetaId != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.MetaId))
		i--
		dAtA[i] = 0x48
	}
	if m.MicroHeight != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.MicroHeight))
		i--
		dAtA[i] = 0x40
	}
	if m.Height != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.Height))
		i--
		dAtA[i] = 0x38
	}
	if m.Index != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x30
	}
	if m.Flagged {
		i--
		if m.Flagged {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	if m.AggregatePower != 0 {
		i = encodeVarintAggregate(dAtA, i, uint64(m.AggregatePower))
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
	if m.AggregatePower != 0 {
		n += 1 + sovAggregate(uint64(m.AggregatePower))
	}
	if m.Flagged {
		n += 2
	}
	if m.Index != 0 {
		n += 1 + sovAggregate(uint64(m.Index))
	}
	if m.Height != 0 {
		n += 1 + sovAggregate(uint64(m.Height))
	}
	if m.MicroHeight != 0 {
		n += 1 + sovAggregate(uint64(m.MicroHeight))
	}
	if m.MetaId != 0 {
		n += 1 + sovAggregate(uint64(m.MetaId))
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
				return fmt.Errorf("proto: wrong wireType = %d for field AggregatePower", wireType)
			}
			m.AggregatePower = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.AggregatePower |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
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
		case 6:
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
		case 7:
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
		case 8:
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
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MetaId", wireType)
			}
			m.MetaId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAggregate
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
