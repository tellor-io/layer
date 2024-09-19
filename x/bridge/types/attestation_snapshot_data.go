package types

import (
	"github.com/gogo/protobuf/proto"
)

// AttestationSnapshots holds the snapshots of attestations.
// Each attestation's snapshots are stored in a slice of bytes.
type AttestationSnapshotData struct {
	ValidatorCheckpoint  []byte `protobuf:"bytes,1,rep,name=validator_checkpoint,proto3"`
	AttestationTimestamp uint64 `protobuf:"varint,2,rep,name=attestation_timestamp,proto3"`
	PrevReportTimestamp  uint64 `protobuf:"varint,3,rep,name=prev_report_timestamp,proto3"`
	NextReportTimestamp  uint64 `protobuf:"varint,4,rep,name=next_report_timestamp,proto3"`
	QueryId              []byte `protobuf:"bytes,5,rep,name=query_id,proto3"`
	Timestamp            uint64 `protobuf:"varint,6,rep,name=timestamp,proto3"`
}

// Ensure AttestationSnapshotData implements proto.Message
var _ proto.Message = &AttestationSnapshotData{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*AttestationSnapshotData) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*AttestationSnapshotData) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *AttestationSnapshotData) String() string {
	return proto.CompactTextString(m)
}
