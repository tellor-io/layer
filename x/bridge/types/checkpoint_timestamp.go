package types

import (
	"github.com/gogo/protobuf/proto"
)

// CheckpointTimestamp wraps a uint64 to be used with the codec
type CheckpointTimestamp struct {
	Timestamp uint64 `protobuf:"varint,1,opt,name=timestamp,proto3"`
}

// Ensure CheckpointTimestamp implements proto.Message
var _ proto.Message = &CheckpointTimestamp{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*CheckpointTimestamp) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*CheckpointTimestamp) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *CheckpointTimestamp) String() string {
	return proto.CompactTextString(m)
}
