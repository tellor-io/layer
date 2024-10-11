package types

import (
	"github.com/gogo/protobuf/proto"
)

type DisputePowerTotals struct {
	Users        uint64 `protobuf:"varint,1,opt,name=users,proto3"`
	Reporters    uint64 `protobuf:"varint,2,opt,name=reporters,proto3"`
	Tokenholders uint64 `protobuf:"varint,3,opt,name=tokenholders,proto3"`
	Multisig     uint64 `protobuf:"varint,4,opt,name=multisig,proto3"`
}

// Ensure WithdrawalId implements proto.Message
var _ proto.Message = &DisputePowerTotals{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*DisputePowerTotals) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*DisputePowerTotals) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *DisputePowerTotals) String() string {
	return proto.CompactTextString(m)
}
