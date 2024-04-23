// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: layer/registry/data_spec.proto

package types

import (
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ABIComponent is a specification for how to interpret abi_components
type ABIComponent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// name
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// type
	FieldType string `protobuf:"bytes,2,opt,name=field_type,json=fieldType,proto3" json:"field_type,omitempty"`
	// consider taking this recursion out and make it once only
	NestedComponent []*ABIComponent `protobuf:"bytes,3,rep,name=nested_component,json=nestedComponent,proto3" json:"nested_component,omitempty"`
}

func (x *ABIComponent) Reset() {
	*x = ABIComponent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_data_spec_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ABIComponent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ABIComponent) ProtoMessage() {}

func (x *ABIComponent) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_data_spec_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ABIComponent.ProtoReflect.Descriptor instead.
func (*ABIComponent) Descriptor() ([]byte, []int) {
	return file_layer_registry_data_spec_proto_rawDescGZIP(), []int{0}
}

func (x *ABIComponent) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ABIComponent) GetFieldType() string {
	if x != nil {
		return x.FieldType
	}
	return ""
}

func (x *ABIComponent) GetNestedComponent() []*ABIComponent {
	if x != nil {
		return x.NestedComponent
	}
	return nil
}

// DataSpec is a specification for how to interpret and aggregate data
type DataSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// ipfs hash of the data spec
	DocumentHash string `protobuf:"bytes,1,opt,name=document_hash,json=documentHash,proto3" json:"document_hash,omitempty"`
	// the value's datatype for decoding the value
	ResponseValueType string `protobuf:"bytes,2,opt,name=response_value_type,json=responseValueType,proto3" json:"response_value_type,omitempty"`
	// the abi components for decoding
	AbiComponents []*ABIComponent `protobuf:"bytes,3,rep,name=abi_components,json=abiComponents,proto3" json:"abi_components,omitempty"`
	// how to aggregate the data (ie. average, median, mode, etc) for aggregating reports and arriving at final value
	AggregationMethod string `protobuf:"bytes,4,opt,name=aggregation_method,json=aggregationMethod,proto3" json:"aggregation_method,omitempty"`
	// address that originally registered the data spec
	Registrar string `protobuf:"bytes,5,opt,name=registrar,proto3" json:"registrar,omitempty"`
	// report_buffer_window specifies the duration of the time window following an initial report
	// during which additional reports can be submitted. This duration acts as a buffer, allowing
	// a collection of related reports in a defined time frame. The window ensures that all
	// pertinent reports are aggregated together before arriving at a final value. This defaults
	// to 0s if not specified.
	// extensions: treat as a golang time.duration, don't allow nil values, don't omit empty values
	ReportBufferWindow *durationpb.Duration `protobuf:"bytes,6,opt,name=report_buffer_window,json=reportBufferWindow,proto3" json:"report_buffer_window,omitempty"`
}

func (x *DataSpec) Reset() {
	*x = DataSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_data_spec_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataSpec) ProtoMessage() {}

func (x *DataSpec) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_data_spec_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataSpec.ProtoReflect.Descriptor instead.
func (*DataSpec) Descriptor() ([]byte, []int) {
	return file_layer_registry_data_spec_proto_rawDescGZIP(), []int{1}
}

func (x *DataSpec) GetDocumentHash() string {
	if x != nil {
		return x.DocumentHash
	}
	return ""
}

func (x *DataSpec) GetResponseValueType() string {
	if x != nil {
		return x.ResponseValueType
	}
	return ""
}

func (x *DataSpec) GetAbiComponents() []*ABIComponent {
	if x != nil {
		return x.AbiComponents
	}
	return nil
}

func (x *DataSpec) GetAggregationMethod() string {
	if x != nil {
		return x.AggregationMethod
	}
	return ""
}

func (x *DataSpec) GetRegistrar() string {
	if x != nil {
		return x.Registrar
	}
	return ""
}

func (x *DataSpec) GetReportBufferWindow() *durationpb.Duration {
	if x != nil {
		return x.ReportBufferWindow
	}
	return nil
}

var File_layer_registry_data_spec_proto protoreflect.FileDescriptor

var file_layer_registry_data_spec_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2f, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x73, 0x70, 0x65, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x1a, 0x11, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2f, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x14,
	0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8a, 0x01, 0x0a, 0x0c, 0x41, 0x42, 0x49, 0x43, 0x6f, 0x6d, 0x70,
	0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x66, 0x69, 0x65,
	0x6c, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x66,
	0x69, 0x65, 0x6c, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x47, 0x0a, 0x10, 0x6e, 0x65, 0x73, 0x74,
	0x65, 0x64, 0x5f, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x41, 0x42, 0x49, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74,
	0x52, 0x0f, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e,
	0x74, 0x22, 0xe7, 0x02, 0x0a, 0x08, 0x44, 0x61, 0x74, 0x61, 0x53, 0x70, 0x65, 0x63, 0x12, 0x23,
	0x0a, 0x0d, 0x64, 0x6f, 0x63, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x64, 0x6f, 0x63, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x48,
	0x61, 0x73, 0x68, 0x12, 0x2e, 0x0a, 0x13, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x5f,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x11, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x43, 0x0a, 0x0e, 0x61, 0x62, 0x69, 0x5f, 0x63, 0x6f, 0x6d, 0x70, 0x6f,
	0x6e, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x41, 0x42, 0x49,
	0x43, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x52, 0x0d, 0x61, 0x62, 0x69, 0x43, 0x6f,
	0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x2d, 0x0a, 0x12, 0x61, 0x67, 0x67, 0x72,
	0x65, 0x67, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x36, 0x0a, 0x09, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x61, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x42, 0x18, 0xd2, 0xb4, 0x2d, 0x14,
	0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x52, 0x09, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61, 0x72, 0x12,
	0x5a, 0x0a, 0x14, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72,
	0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x0d, 0xc8, 0xde, 0x1f, 0x00, 0x98, 0xdf,
	0x1f, 0x01, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52, 0x12, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x42,
	0x75, 0x66, 0x66, 0x65, 0x72, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x42, 0x2d, 0x5a, 0x2b, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x65, 0x6c, 0x6c, 0x6f, 0x72,
	0x2d, 0x69, 0x6f, 0x2f, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x78, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_layer_registry_data_spec_proto_rawDescOnce sync.Once
	file_layer_registry_data_spec_proto_rawDescData = file_layer_registry_data_spec_proto_rawDesc
)

func file_layer_registry_data_spec_proto_rawDescGZIP() []byte {
	file_layer_registry_data_spec_proto_rawDescOnce.Do(func() {
		file_layer_registry_data_spec_proto_rawDescData = protoimpl.X.CompressGZIP(file_layer_registry_data_spec_proto_rawDescData)
	})
	return file_layer_registry_data_spec_proto_rawDescData
}

var file_layer_registry_data_spec_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_layer_registry_data_spec_proto_goTypes = []interface{}{
	(*ABIComponent)(nil),        // 0: layer.registry.ABIComponent
	(*DataSpec)(nil),            // 1: layer.registry.DataSpec
	(*durationpb.Duration)(nil), // 2: google.protobuf.Duration
}
var file_layer_registry_data_spec_proto_depIdxs = []int32{
	0, // 0: layer.registry.ABIComponent.nested_component:type_name -> layer.registry.ABIComponent
	0, // 1: layer.registry.DataSpec.abi_components:type_name -> layer.registry.ABIComponent
	2, // 2: layer.registry.DataSpec.report_buffer_window:type_name -> google.protobuf.Duration
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_layer_registry_data_spec_proto_init() }
func file_layer_registry_data_spec_proto_init() {
	if File_layer_registry_data_spec_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_layer_registry_data_spec_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ABIComponent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_layer_registry_data_spec_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataSpec); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_layer_registry_data_spec_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_layer_registry_data_spec_proto_goTypes,
		DependencyIndexes: file_layer_registry_data_spec_proto_depIdxs,
		MessageInfos:      file_layer_registry_data_spec_proto_msgTypes,
	}.Build()
	File_layer_registry_data_spec_proto = out.File
	file_layer_registry_data_spec_proto_rawDesc = nil
	file_layer_registry_data_spec_proto_goTypes = nil
	file_layer_registry_data_spec_proto_depIdxs = nil
}