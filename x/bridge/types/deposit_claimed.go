package types

import (
	"github.com/gogo/protobuf/proto"
)

// DepositClaimed holds the claimed status of a deposit ID.
type DepositClaimed struct {
	Claimed bool `protobuf:"bool,1,opt,name=claimed,proto3"`
}

// Ensure DepositIdClaimed implements proto.Message
var _ proto.Message = &DepositClaimed{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*DepositClaimed) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*DepositClaimed) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *DepositClaimed) String() string {
	return proto.CompactTextString(m)
}
