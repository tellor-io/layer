package types

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
)

type AttestationRequest struct {
	Snapshot []byte `protobuf:"bytes,1,opt,name=snapshot,proto3"`
}

// AttestationRequests holds requests for attestations.
type AttestationRequests struct {
	Requests []*AttestationRequest `protobuf:"bytes,1,rep,name=requests,proto3"`
}

// AddRequest adds a request for an attestation to the list of requests.
// `request` is the request for an attestation.
func (b *AttestationRequests) AddRequest(request *AttestationRequest) {
	b.Requests = append(b.Requests, request)
}

// HasSnapshot returns true if the given snapshot already exists in requests.
func (b *AttestationRequests) HasSnapshot(snapshot []byte) bool {
	for _, request := range b.Requests {
		if bytes.Equal(request.Snapshot, snapshot) {
			return true
		}
	}
	return false
}

// Ensure OracleAttestations implements proto.Message
var _ proto.Message = &AttestationRequests{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*AttestationRequests) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*AttestationRequests) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *AttestationRequests) String() string {
	return proto.CompactTextString(m)
}
