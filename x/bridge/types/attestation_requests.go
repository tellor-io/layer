package types

import (
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