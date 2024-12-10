package keeper

import (
	collcodec "cosmossdk.io/collections/codec"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/gogo/protobuf/proto"
	"github.com/tellor-io/layer/x/oracle/types"
)

func AggregateValueCodec(cdc codec.BinaryCodec) collcodec.ValueCodec[types.Aggregate] {
	return collcodec.NewAltValueCodec(codec.CollValue[types.Aggregate](cdc), func(bytes []byte) (types.Aggregate, error) {
		var agg AggregateLegacy
		err := cdc.Unmarshal(bytes, &agg)
		if err != nil {
			return types.Aggregate{}, err
		}
		return types.Aggregate{
			QueryId:           agg.QueryId,
			AggregateValue:    agg.AggregateValue,
			AggregateReporter: agg.AggregateReporter,
			AggregatePower:    agg.ReporterPower,
			Flagged:           agg.Flagged,
			Index:             agg.Index,
			Height:            agg.Height,
			MicroHeight:       agg.MicroHeight,
		}, nil
	})
}

type AggregateLegacy struct {
	// query_id is the id of the query
	QueryId []byte `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
	// aggregate_value is the value of the aggregate
	AggregateValue string `protobuf:"bytes,2,opt,name=aggregate_value,json=aggregateValue,proto3" json:"aggregate_value,omitempty"`
	// aggregate_reporter is the address of the reporter
	AggregateReporter string `protobuf:"bytes,3,opt,name=aggregate_reporter,json=aggregateReporter,proto3" json:"aggregate_reporter,omitempty"`
	// reporter_power is the power of the reporter
	ReporterPower uint64 `protobuf:"varint,4,opt,name=reporter_power,json=reporterPower,proto3" json:"reporter_power,omitempty"`
	// list of reporters that were included in the aggregate
	Reporters []*types.AggregateReporter `protobuf:"bytes,6,rep,name=reporters,proto3" json:"reporters,omitempty"`
	// flagged is true if the aggregate was flagged by a dispute
	Flagged bool `protobuf:"varint,7,opt,name=flagged,proto3" json:"flagged,omitempty"`
	// index is the index of the aggregate
	Index uint64 `protobuf:"varint,8,opt,name=index,proto3" json:"index,omitempty"`
	// aggregate_report_index is the index of the aggregate report in the micro reports
	AggregateReportIndex uint64 `protobuf:"varint,9,opt,name=aggregate_report_index,json=aggregateReportIndex,proto3" json:"aggregate_report_index,omitempty"`
	// height of the aggregate report
	Height uint64 `protobuf:"varint,10,opt,name=height,proto3" json:"height,omitempty"`
	// height of the micro report
	MicroHeight uint64 `protobuf:"varint,11,opt,name=micro_height,json=microHeight,proto3" json:"micro_height,omitempty"`
	// meta_id is the id of the querymeta iterator
	MetaId uint64 `protobuf:"varint,12,opt,name=meta_id,json=metaId,proto3" json:"meta_id,omitempty"`
}

var _ proto.Message = &AggregateLegacy{}

func (*AggregateLegacy) Reset() {}
func (m *AggregateLegacy) String() string {
	return proto.CompactTextString(m)
}
func (*AggregateLegacy) ProtoMessage() {}
