package types

import (
	"github.com/gogo/protobuf/proto"
)

// BridgeValsetSignatures holds the signatures of validators.
// Each validator's signatures are stored in a slice of bytes.
type BridgeValsetSignatures struct {
	Signatures [][]byte `protobuf:"bytes,1,rep,name=signatures,proto3"`
}

// NewBridgeValsetSignatures initializes a BridgeValsetSignatures with a given size.
func NewBridgeValsetSignatures(valsetSize int) *BridgeValsetSignatures {
	signatures := make([][]byte, valsetSize)
	for i := range signatures {
		signatures[i] = make([]byte, 0) // Initialize with empty slice, adjust according to needs
	}
	return &BridgeValsetSignatures{Signatures: signatures}
}

// SetSignature sets a signature for a validator at the given index.
// `validatorIndex` is the position of the validator in the bridge valset.
// `signature` is the validator's signature.
// Note: Ensure `validatorIndex` is within bounds before calling this method.
func (b *BridgeValsetSignatures) SetSignature(validatorIndex int, signature []byte) {
	if validatorIndex >= 0 && validatorIndex < len(b.Signatures) {
		b.Signatures[validatorIndex] = signature
	}
}

// Ensure BridgeValsetSignatures implements proto.Message
var _ proto.Message = &BridgeValsetSignatures{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*BridgeValsetSignatures) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*BridgeValsetSignatures) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *BridgeValsetSignatures) String() string {
	return proto.CompactTextString(m)
}
