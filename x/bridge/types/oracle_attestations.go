package types

import (
	"github.com/gogo/protobuf/proto"
)

// OracleAttestations holds the attestations of validators.
// Each validator's attestations are stored in a slice of bytes.
type OracleAttestations struct {
	Attestations [][]byte `protobuf:"bytes,1,rep,name=attestations,proto3"`
}

// NewOracleAttestations initializes a OracleAttestations with a given size.
func NewOracleAttestations(valsetSize int) *OracleAttestations {
	attestations := make([][]byte, valsetSize)
	for i := range attestations {
		attestations[i] = make([]byte, 0) // Initialize with empty slice, adjust according to needs
	}
	return &OracleAttestations{Attestations: attestations}
}

// SetAttestation sets an attestation for a validator at the given index.
// `validatorIndex` is the position of the validator in the bridge valset.
// `attestation` is the validator's signature.
// Note: Ensure `validatorIndex` is within bounds before calling this method.
func (b *OracleAttestations) SetAttestation(validatorIndex int, attestation []byte) {
	if validatorIndex >= 0 && validatorIndex < len(b.Attestations) {
		b.Attestations[validatorIndex] = attestation
	}
}

// Ensure OracleAttestations implements proto.Message
var _ proto.Message = &OracleAttestations{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*OracleAttestations) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*OracleAttestations) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *OracleAttestations) String() string {
	return proto.CompactTextString(m)
}
