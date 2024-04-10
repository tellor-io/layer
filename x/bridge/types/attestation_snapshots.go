package types

import (
	"github.com/gogo/protobuf/proto"
)

// AttestationSnapshots holds the snapshots of attestations.
// Each attestation's snapshots are stored in a slice of bytes.
type AttestationSnapshots struct {
	Snapshots [][]byte `protobuf:"bytes,1,rep,name=snapshots,proto3"`
}

// NewAttestationSnapshots initializes a AttestationSnapshots with a given size.
func NewAttestationSnapshots() *AttestationSnapshots {
	snapshots := make([][]byte, 0) // Initialize with empty slice, adjust according to needs
	return &AttestationSnapshots{Snapshots: snapshots}
}

// SetSnapshot appends an attestation snapshot to the list of snapshots for a given aggregate report.
// `snapshot` is the attestation's snapshot.
func (b *AttestationSnapshots) SetSnapshot(snapshot []byte) {
	b.Snapshots = append(b.Snapshots, snapshot)
}

// Ensure AttestationSnapshots implements proto.Message
var _ proto.Message = &AttestationSnapshots{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*AttestationSnapshots) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*AttestationSnapshots) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *AttestationSnapshots) String() string {
	return proto.CompactTextString(m)
}
