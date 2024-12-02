package types

import (
	"github.com/gogo/protobuf/proto"
)

// SnapshotLimit holds the limit of number of attestation requests per block.
type SnapshotLimit struct {
	Limit uint64 `protobuf:"varint,1,opt,name=limit,proto3"`
}

// Ensure SnapshotLimit implements proto.Message
var _ proto.Message = &SnapshotLimit{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*SnapshotLimit) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*SnapshotLimit) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *SnapshotLimit) String() string {
	return proto.CompactTextString(m)
}
