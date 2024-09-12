package types

import (
	"github.com/gogo/protobuf/proto"
)

// ValidatorCheckpointParams holds the signatures of validators.
// Each validator's signatures are stored in a slice of bytes.
type ValidatorCheckpointParams struct {
	Checkpoint     []byte `protobuf:"bytes,1,opt,name=checkpoint,proto3"`
	ValsetHash     []byte `protobuf:"bytes,2,opt,name=valset_hash,proto3"`
	Timestamp      uint64  `protobuf:"varint,3,opt,name=timestamp,proto3"`
	PowerThreshold uint64  `protobuf:"varint,4,opt,name=power_threshold,proto3"`
}

// Ensure ValidatorCheckpointParams implements proto.Message
var _ proto.Message = &ValidatorCheckpointParams{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*ValidatorCheckpointParams) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*ValidatorCheckpointParams) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *ValidatorCheckpointParams) String() string {
	return proto.CompactTextString(m)
}
