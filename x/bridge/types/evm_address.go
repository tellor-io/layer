package types

import (
	"github.com/gogo/protobuf/proto"
)

// EVMAddress wraps a [32]byte to be used with the codec
type EVMAddress struct {
	EVMAddress []byte `protobuf:"bytes,1,opt,name=evm_address,proto3,casttype=[32]byte"`
}

// Ensure EVMAddress implements proto.Message
var _ proto.Message = &EVMAddress{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*EVMAddress) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*EVMAddress) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *EVMAddress) String() string {
	return proto.CompactTextString(m)
}
