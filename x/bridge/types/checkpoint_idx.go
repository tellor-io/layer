package types

import (
	"github.com/gogo/protobuf/proto"
)

// CheckpointIdx wraps a uint64 to be used with the codec
type CheckpointIdx struct {
	Index uint64 `protobuf:"varint,1,opt,name=timestamp,proto3"`
}

// Ensure CheckpointIdx implements proto.Message
var _ proto.Message = &CheckpointIdx{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*CheckpointIdx) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*CheckpointIdx) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *CheckpointIdx) String() string {
	return proto.CompactTextString(m)
}
