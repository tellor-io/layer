package types

import (
	"github.com/gogo/protobuf/proto"
)

// CheckpointTimestamp wraps a uint64 to be used with the codec
type ValidatorSetTimestamps struct {
	Timestamps []ValidatorSetTimestamp `protobuf:"bytes,1,rep,name=timestamps"`
}

type ValidatorSetTimestamp struct {
	Timestamp uint64 `protobuf:"varint,1,opt,name=timestamp,proto3"`
}

// Ensure CheckpointTimestamp implements proto.Message
var _ proto.Message = &ValidatorSetTimestamps{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*ValidatorSetTimestamps) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*ValidatorSetTimestamps) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *ValidatorSetTimestamps) String() string {
	return proto.CompactTextString(m)
}
