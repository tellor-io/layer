package types

import (
	"github.com/gogo/protobuf/proto"
)

// QueryDataLimit holds the limit of query data bytes per micro report.
type QueryDataLimit struct {
	Limit uint64 `protobuf:"varint,1,opt,name=limit,proto3"`
}

// Ensure QueryDataLimit implements proto.Message
var _ proto.Message = &QueryDataLimit{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*QueryDataLimit) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*QueryDataLimit) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *QueryDataLimit) String() string {
	return proto.CompactTextString(m)
}

func InitialQueryDataLimit() uint64 {
	return 524288
}
