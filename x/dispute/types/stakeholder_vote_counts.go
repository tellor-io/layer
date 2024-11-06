package types

import (
	"github.com/gogo/protobuf/proto"
)

type StakeholderVoteCounts struct {
	Users        VoteCounts `protobuf:"bytes,1,opt,name=users,proto3"`
	Reporters    VoteCounts `protobuf:"bytes,2,opt,name=reporters,proto3"`
	Tokenholders VoteCounts `protobuf:"bytes,3,opt,name=tokenholders,proto3"`
	Team         VoteCounts `protobuf:"bytes,4,opt,name=team,proto3"`
}

// Ensure WithdrawalId implements proto.Message
var _ proto.Message = &StakeholderVoteCounts{}

// ProtoMessage is a no-op method to satisfy the proto.Message interface
func (*StakeholderVoteCounts) ProtoMessage() {}

// Reset is a no-op method to satisfy the proto.Message interface
func (*StakeholderVoteCounts) Reset() {}

// String returns a string representation, satisfying the proto.Message interface
func (m *StakeholderVoteCounts) String() string {
	return proto.CompactTextString(m)
}
