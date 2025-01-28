package types

import (
	"github.com/gogo/protobuf/proto"
)

// EVMAddressRegistered wraps a bool to be used with the codec
type EVMAddressRegistered struct {
	Registered bool `protobuf:"varint,1,opt,name=registered"`
}

// Ensure EVMAddressRegistered implements proto.Message
var _ proto.Message = &EVMAddressRegistered{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*EVMAddressRegistered) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*EVMAddressRegistered) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *EVMAddressRegistered) String() string {
	return proto.CompactTextString(m)
}
