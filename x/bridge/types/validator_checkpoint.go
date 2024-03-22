package types

import (
	"github.com/gogo/protobuf/proto"
)

// ValidatorCheckpoint wraps a [32]byte to be used with the codec
type ValidatorCheckpoint struct {
	Checkpoint []byte `protobuf:"bytes,1,opt,name=checkpoint,proto3,casttype=[32]byte"`
}

// Ensure ValidatorCheckpoint implements proto.Message
var _ proto.Message = &ValidatorCheckpoint{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*ValidatorCheckpoint) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*ValidatorCheckpoint) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *ValidatorCheckpoint) String() string {
	return proto.CompactTextString(m)
}
