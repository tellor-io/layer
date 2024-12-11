// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: layer/oracle/query_meta.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
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

// QueryMeta represents the metadata of a query
type QueryMeta struct {
	// unique id of the query that changes after query's lifecycle ends
	Id uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// amount of tokens that was tipped
	Amount cosmossdk_io_math.Int `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
	// expiration time of the query
	Expiration uint64 `protobuf:"varint,3,opt,name=expiration,proto3" json:"expiration,omitempty"`
	// timeframe of the query according to the data spec
	RegistrySpecBlockWindow uint64 `protobuf:"varint,4,opt,name=registry_spec_block_window,json=registrySpecBlockWindow,proto3" json:"registry_spec_block_window,omitempty"`
	// indicates whether query has revealed reports
	HasRevealedReports bool `protobuf:"varint,5,opt,name=has_revealed_reports,json=hasRevealedReports,proto3" json:"has_revealed_reports,omitempty"`
	// query_data: decodable bytes to field of the data spec
	QueryData []byte `protobuf:"bytes,6,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
	// string identifier of the data spec
	QueryType string `protobuf:"bytes,7,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
	// bool cycle list query
	CycleList bool `protobuf:"varint,8,opt,name=cycle_list,json=cycleList,proto3" json:"cycle_list,omitempty"`
}

func (m *QueryMeta) Reset()         { *m = QueryMeta{} }
func (m *QueryMeta) String() string { return proto.CompactTextString(m) }
func (*QueryMeta) ProtoMessage()    {}
func (*QueryMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_072f14e329c22246, []int{0}
}
func (m *QueryMeta) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryMeta.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryMeta.Merge(m, src)
}
func (m *QueryMeta) XXX_Size() int {
	return m.Size()
}
func (m *QueryMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryMeta.DiscardUnknown(m)
}

var xxx_messageInfo_QueryMeta proto.InternalMessageInfo

func (m *QueryMeta) GetId() uint64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *QueryMeta) GetExpiration() uint64 {
	if m != nil {
		return m.Expiration
	}
	return 0
}

func (m *QueryMeta) GetRegistrySpecBlockWindow() uint64 {
	if m != nil {
		return m.RegistrySpecBlockWindow
	}
	return 0
}

func (m *QueryMeta) GetHasRevealedReports() bool {
	if m != nil {
		return m.HasRevealedReports
	}
	return false
}

func (m *QueryMeta) GetQueryData() []byte {
	if m != nil {
		return m.QueryData
	}
	return nil
}

func (m *QueryMeta) GetQueryType() string {
	if m != nil {
		return m.QueryType
	}
	return ""
}

func (m *QueryMeta) GetCycleList() bool {
	if m != nil {
		return m.CycleList
	}
	return false
}

type Reward struct {
	TotalPower uint64                      `protobuf:"varint,1,opt,name=totalPower,proto3" json:"totalPower,omitempty"`
	Amount     cosmossdk_io_math.LegacyDec `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"amount"`
	// cycle_list indicates if a reward should also include timebasedRewards(tbr)
	CycleList bool `protobuf:"varint,3,opt,name=cycle_list,json=cycleList,proto3" json:"cycle_list,omitempty"`
	// if cyclist then tbr amount can be fetched by this height
	BlockHeight   uint64                      `protobuf:"varint,4,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	PowerPaidOut  cosmossdk_io_math.LegacyDec `protobuf:"bytes,5,opt,name=power_paid_out,json=powerPaidOut,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"power_paid_out"`
	AmountPaidOut cosmossdk_io_math.LegacyDec `protobuf:"bytes,6,opt,name=amount_paid_out,json=amountPaidOut,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"amount_paid_out"`
}

func (m *Reward) Reset()         { *m = Reward{} }
func (m *Reward) String() string { return proto.CompactTextString(m) }
func (*Reward) ProtoMessage()    {}
func (*Reward) Descriptor() ([]byte, []int) {
	return fileDescriptor_072f14e329c22246, []int{1}
}
func (m *Reward) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Reward) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Reward.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Reward) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Reward.Merge(m, src)
}
func (m *Reward) XXX_Size() int {
	return m.Size()
}
func (m *Reward) XXX_DiscardUnknown() {
	xxx_messageInfo_Reward.DiscardUnknown(m)
}

var xxx_messageInfo_Reward proto.InternalMessageInfo

func (m *Reward) GetTotalPower() uint64 {
	if m != nil {
		return m.TotalPower
	}
	return 0
}

func (m *Reward) GetCycleList() bool {
	if m != nil {
		return m.CycleList
	}
	return false
}

func (m *Reward) GetBlockHeight() uint64 {
	if m != nil {
		return m.BlockHeight
	}
	return 0
}

func init() {
	proto.RegisterType((*QueryMeta)(nil), "layer.oracle.QueryMeta")
	proto.RegisterType((*Reward)(nil), "layer.oracle.reward")
}

func init() { proto.RegisterFile("layer/oracle/query_meta.proto", fileDescriptor_072f14e329c22246) }

var fileDescriptor_072f14e329c22246 = []byte{
	// 535 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x93, 0xc1, 0x6e, 0xd3, 0x30,
	0x1c, 0xc6, 0x9b, 0x6e, 0x94, 0xd5, 0x94, 0x21, 0xac, 0x21, 0xc2, 0xd0, 0xb2, 0xb2, 0x53, 0x11,
	0x5a, 0x03, 0xe2, 0xc8, 0xad, 0xeb, 0x81, 0x4a, 0x43, 0x8c, 0x80, 0x34, 0xc1, 0x25, 0x72, 0x93,
	0x3f, 0x89, 0xb5, 0x24, 0x0e, 0xf6, 0x3f, 0x74, 0x79, 0x0b, 0x1e, 0x86, 0x87, 0xd8, 0x81, 0xc3,
	0xc4, 0x09, 0x71, 0x98, 0x50, 0x7b, 0xe2, 0x2d, 0x50, 0x6c, 0x8f, 0x95, 0x72, 0xdb, 0x25, 0x8a,
	0xbf, 0xdf, 0x3f, 0x5f, 0x3e, 0x7f, 0x96, 0xc9, 0x4e, 0xc6, 0x6a, 0x90, 0xbe, 0x90, 0x2c, 0xca,
	0xc0, 0xff, 0x54, 0x81, 0xac, 0xc3, 0x1c, 0x90, 0x0d, 0x4b, 0x29, 0x50, 0xd0, 0x9e, 0xc6, 0x43,
	0x83, 0xb7, 0x1f, 0x44, 0x42, 0xe5, 0x42, 0x85, 0x9a, 0xf9, 0x66, 0x61, 0x06, 0xb7, 0xb7, 0x12,
	0x91, 0x08, 0xa3, 0x37, 0x6f, 0x56, 0xf5, 0x12, 0x21, 0x92, 0x0c, 0x7c, 0xbd, 0x9a, 0x56, 0x1f,
	0xfd, 0xb8, 0x92, 0x0c, 0xb9, 0x28, 0x2c, 0xdf, 0x5d, 0xe5, 0xc8, 0x73, 0x50, 0xc8, 0xf2, 0xd2,
	0x0e, 0xdc, 0x65, 0x39, 0x2f, 0x84, 0xaf, 0x9f, 0x46, 0xda, 0xfb, 0xd6, 0x26, 0xdd, 0x37, 0x4d,
	0xce, 0x57, 0x80, 0x8c, 0x6e, 0x92, 0x36, 0x8f, 0x5d, 0xa7, 0xef, 0x0c, 0xd6, 0x83, 0x36, 0x8f,
	0xe9, 0x01, 0xe9, 0xb0, 0x5c, 0x54, 0x05, 0xba, 0xed, 0xbe, 0x33, 0xe8, 0x8e, 0x9e, 0x9c, 0x5d,
	0xec, 0xb6, 0x7e, 0x5e, 0xec, 0xde, 0x33, 0x69, 0x55, 0x7c, 0x32, 0xe4, 0xc2, 0xcf, 0x19, 0xa6,
	0xc3, 0x49, 0x81, 0xdf, 0xbf, 0xee, 0x13, 0xbb, 0x8d, 0x49, 0x81, 0x81, 0xfd, 0x94, 0x7a, 0x84,
	0xc0, 0x69, 0xc9, 0x4d, 0x54, 0x77, 0x4d, 0x9b, 0x2f, 0x29, 0xf4, 0x05, 0xd9, 0x96, 0x90, 0x70,
	0x85, 0xb2, 0x0e, 0x55, 0x09, 0x51, 0x38, 0xcd, 0x44, 0x74, 0x12, 0xce, 0x78, 0x11, 0x8b, 0x99,
	0xbb, 0xae, 0xe7, 0xef, 0x5f, 0x4e, 0xbc, 0x2d, 0x21, 0x1a, 0x35, 0xfc, 0x58, 0x63, 0xfa, 0x94,
	0x6c, 0xa5, 0x4c, 0x85, 0x12, 0x3e, 0x03, 0xcb, 0x20, 0x0e, 0x25, 0x94, 0x42, 0xa2, 0x72, 0x6f,
	0xf4, 0x9d, 0xc1, 0x46, 0x40, 0x53, 0xa6, 0x02, 0x8b, 0x02, 0x43, 0xe8, 0x0e, 0x21, 0xe6, 0x60,
	0x62, 0x86, 0xcc, 0xed, 0xf4, 0x9d, 0x41, 0x2f, 0xe8, 0x6a, 0x65, 0xcc, 0x90, 0x5d, 0x61, 0xac,
	0x4b, 0x70, 0x6f, 0x36, 0xdb, 0xb6, 0xf8, 0x5d, 0x5d, 0x42, 0x83, 0xa3, 0x3a, 0xca, 0x20, 0xcc,
	0xb8, 0x42, 0x77, 0x43, 0xff, 0xa5, 0xab, 0x95, 0x43, 0xae, 0x70, 0xef, 0x77, 0x9b, 0x74, 0x24,
	0xcc, 0x98, 0x8c, 0x9b, 0x6d, 0xa3, 0x40, 0x96, 0x1d, 0x89, 0x19, 0x48, 0xdb, 0xe9, 0x92, 0x42,
	0x27, 0x2b, 0xdd, 0x3e, 0xb3, 0xdd, 0x3e, 0xfc, 0xbf, 0xdb, 0x43, 0x48, 0x58, 0x54, 0x8f, 0x21,
	0x5a, 0x6a, 0x78, 0x0c, 0xd1, 0xdf, 0x86, 0xff, 0x0d, 0xb5, 0xb6, 0x12, 0x8a, 0x3e, 0x22, 0x3d,
	0x53, 0x69, 0x0a, 0x3c, 0x49, 0xd1, 0x56, 0x7a, 0x4b, 0x6b, 0x2f, 0xb5, 0x44, 0x8f, 0xc9, 0x66,
	0xd9, 0xa4, 0x0a, 0x4b, 0xc6, 0xe3, 0x50, 0x54, 0xa8, 0x0b, 0xbc, 0x56, 0xa8, 0x9e, 0x36, 0x3a,
	0x62, 0x3c, 0x7e, 0x5d, 0x21, 0x7d, 0x4f, 0xee, 0x98, 0x90, 0x57, 0xce, 0x9d, 0xeb, 0x3a, 0xdf,
	0x36, 0x4e, 0xd6, 0x7a, 0x74, 0x70, 0x36, 0xf7, 0x9c, 0xf3, 0xb9, 0xe7, 0xfc, 0x9a, 0x7b, 0xce,
	0x97, 0x85, 0xd7, 0x3a, 0x5f, 0x78, 0xad, 0x1f, 0x0b, 0xaf, 0xf5, 0xe1, 0x71, 0xc2, 0x31, 0xad,
	0xa6, 0xc3, 0x48, 0xe4, 0x3e, 0x42, 0x96, 0x09, 0xb9, 0xcf, 0x85, 0x6f, 0xee, 0xe6, 0xe9, 0xe5,
	0xed, 0x6c, 0xce, 0x57, 0x4d, 0x3b, 0xfa, 0x1a, 0x3c, 0xff, 0x13, 0x00, 0x00, 0xff, 0xff, 0x12,
	0xd6, 0x2c, 0xdb, 0xba, 0x03, 0x00, 0x00,
}

func (m *QueryMeta) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryMeta) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryMeta) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.CycleList {
		i--
		if m.CycleList {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x40
	}
	if len(m.QueryType) > 0 {
		i -= len(m.QueryType)
		copy(dAtA[i:], m.QueryType)
		i = encodeVarintQueryMeta(dAtA, i, uint64(len(m.QueryType)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.QueryData) > 0 {
		i -= len(m.QueryData)
		copy(dAtA[i:], m.QueryData)
		i = encodeVarintQueryMeta(dAtA, i, uint64(len(m.QueryData)))
		i--
		dAtA[i] = 0x32
	}
	if m.HasRevealedReports {
		i--
		if m.HasRevealedReports {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	if m.RegistrySpecBlockWindow != 0 {
		i = encodeVarintQueryMeta(dAtA, i, uint64(m.RegistrySpecBlockWindow))
		i--
		dAtA[i] = 0x20
	}
	if m.Expiration != 0 {
		i = encodeVarintQueryMeta(dAtA, i, uint64(m.Expiration))
		i--
		dAtA[i] = 0x18
	}
	{
		size := m.Amount.Size()
		i -= size
		if _, err := m.Amount.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintQueryMeta(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.Id != 0 {
		i = encodeVarintQueryMeta(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Reward) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Reward) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Reward) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.AmountPaidOut.Size()
		i -= size
		if _, err := m.AmountPaidOut.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintQueryMeta(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x32
	{
		size := m.PowerPaidOut.Size()
		i -= size
		if _, err := m.PowerPaidOut.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintQueryMeta(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if m.BlockHeight != 0 {
		i = encodeVarintQueryMeta(dAtA, i, uint64(m.BlockHeight))
		i--
		dAtA[i] = 0x20
	}
	if m.CycleList {
		i--
		if m.CycleList {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x18
	}
	{
		size := m.Amount.Size()
		i -= size
		if _, err := m.Amount.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintQueryMeta(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if m.TotalPower != 0 {
		i = encodeVarintQueryMeta(dAtA, i, uint64(m.TotalPower))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintQueryMeta(dAtA []byte, offset int, v uint64) int {
	offset -= sovQueryMeta(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *QueryMeta) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovQueryMeta(uint64(m.Id))
	}
	l = m.Amount.Size()
	n += 1 + l + sovQueryMeta(uint64(l))
	if m.Expiration != 0 {
		n += 1 + sovQueryMeta(uint64(m.Expiration))
	}
	if m.RegistrySpecBlockWindow != 0 {
		n += 1 + sovQueryMeta(uint64(m.RegistrySpecBlockWindow))
	}
	if m.HasRevealedReports {
		n += 2
	}
	l = len(m.QueryData)
	if l > 0 {
		n += 1 + l + sovQueryMeta(uint64(l))
	}
	l = len(m.QueryType)
	if l > 0 {
		n += 1 + l + sovQueryMeta(uint64(l))
	}
	if m.CycleList {
		n += 2
	}
	return n
}

func (m *Reward) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.TotalPower != 0 {
		n += 1 + sovQueryMeta(uint64(m.TotalPower))
	}
	l = m.Amount.Size()
	n += 1 + l + sovQueryMeta(uint64(l))
	if m.CycleList {
		n += 2
	}
	if m.BlockHeight != 0 {
		n += 1 + sovQueryMeta(uint64(m.BlockHeight))
	}
	l = m.PowerPaidOut.Size()
	n += 1 + l + sovQueryMeta(uint64(l))
	l = m.AmountPaidOut.Size()
	n += 1 + l + sovQueryMeta(uint64(l))
	return n
}

func sovQueryMeta(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozQueryMeta(x uint64) (n int) {
	return sovQueryMeta(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *QueryMeta) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQueryMeta
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
			return fmt.Errorf("proto: QueryMeta: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryMeta: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Id |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
				return ErrInvalidLengthQueryMeta
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueryMeta
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Amount.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Expiration", wireType)
			}
			m.Expiration = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Expiration |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RegistrySpecBlockWindow", wireType)
			}
			m.RegistrySpecBlockWindow = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RegistrySpecBlockWindow |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field HasRevealedReports", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
			m.HasRevealedReports = bool(v != 0)
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryData", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
				return ErrInvalidLengthQueryMeta
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthQueryMeta
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryData = append(m.QueryData[:0], dAtA[iNdEx:postIndex]...)
			if m.QueryData == nil {
				m.QueryData = []byte{}
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field QueryType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
				return ErrInvalidLengthQueryMeta
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueryMeta
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.QueryType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CycleList", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
			m.CycleList = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipQueryMeta(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQueryMeta
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
func (m *Reward) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQueryMeta
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
			return fmt.Errorf("proto: reward: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: reward: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalPower", wireType)
			}
			m.TotalPower = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalPower |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
				return ErrInvalidLengthQueryMeta
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueryMeta
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Amount.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CycleList", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
			m.CycleList = bool(v != 0)
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockHeight", wireType)
			}
			m.BlockHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BlockHeight |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PowerPaidOut", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
				return ErrInvalidLengthQueryMeta
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueryMeta
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.PowerPaidOut.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AmountPaidOut", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQueryMeta
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
				return ErrInvalidLengthQueryMeta
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQueryMeta
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.AmountPaidOut.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQueryMeta(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQueryMeta
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
func skipQueryMeta(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQueryMeta
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
					return 0, ErrIntOverflowQueryMeta
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
					return 0, ErrIntOverflowQueryMeta
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
				return 0, ErrInvalidLengthQueryMeta
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQueryMeta
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQueryMeta
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQueryMeta        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQueryMeta          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQueryMeta = fmt.Errorf("proto: unexpected end of group")
)
