package types

import (
	"github.com/gogo/protobuf/proto"
)

// OperatorAddress wraps a [32]byte to be used with the codec
type OperatorAddress struct {
	OperatorAddress []byte `protobuf:"bytes,1,opt,name=operator_address,proto3,casttype=[32]byte"`
}

// Ensure OperatorAddress implements proto.Message
var _ proto.Message = &OperatorAddress{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*OperatorAddress) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*OperatorAddress) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *OperatorAddress) String() string {
	return proto.CompactTextString(m)
}
