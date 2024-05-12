package types

import (
	"github.com/gogo/protobuf/proto"
)

// WithdrawalId holds the latest token bridge withdrawal id.
type WithdrawalId struct {
	Id uint64 `protobuf:"varint,1,opt,name=id,proto3"`
}

// Ensure WithdrawalId implements proto.Message
var _ proto.Message = &WithdrawalId{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*WithdrawalId) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*WithdrawalId) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *WithdrawalId) String() string {
	return proto.CompactTextString(m)
}
