// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: layer/registry/query.proto

package types

import (
	_ "github.com/cosmos/gogoproto/gogoproto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// QueryParamsRequest is request type for the Query/Params RPC method.
type QueryParamsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *QueryParamsRequest) Reset() {
	*x = QueryParamsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryParamsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryParamsRequest) ProtoMessage() {}

func (x *QueryParamsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryParamsRequest.ProtoReflect.Descriptor instead.
func (*QueryParamsRequest) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{0}
}

// QueryParamsResponse is response type for the Query/Params RPC method.
type QueryParamsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// params holds all the parameters of this module.
	Params *Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params,omitempty"`
}

func (x *QueryParamsResponse) Reset() {
	*x = QueryParamsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryParamsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryParamsResponse) ProtoMessage() {}

func (x *QueryParamsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryParamsResponse.ProtoReflect.Descriptor instead.
func (*QueryParamsResponse) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{1}
}

func (x *QueryParamsResponse) GetParams() *Params {
	if x != nil {
		return x.Params
	}
	return nil
}

// QueryGetDataSpecRequest is request type for the Query/GetDataSpec RPC method.
type QueryGetDataSpecRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// queryType is the key to fetch a the corresponding data spec.
	QueryType string `protobuf:"bytes,1,opt,name=query_type,json=queryType,proto3" json:"query_type,omitempty"`
}

func (x *QueryGetDataSpecRequest) Reset() {
	*x = QueryGetDataSpecRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryGetDataSpecRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryGetDataSpecRequest) ProtoMessage() {}

func (x *QueryGetDataSpecRequest) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryGetDataSpecRequest.ProtoReflect.Descriptor instead.
func (*QueryGetDataSpecRequest) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{2}
}

func (x *QueryGetDataSpecRequest) GetQueryType() string {
	if x != nil {
		return x.QueryType
	}
	return ""
}

// QueryGetDataSpecResponse is response type for the Query/GetDataSpec RPC method.
type QueryGetDataSpecResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// spec is the data spec corresponding to the query type.
	Spec *DataSpec `protobuf:"bytes,1,opt,name=spec,proto3" json:"spec,omitempty"`
}

func (x *QueryGetDataSpecResponse) Reset() {
	*x = QueryGetDataSpecResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryGetDataSpecResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryGetDataSpecResponse) ProtoMessage() {}

func (x *QueryGetDataSpecResponse) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryGetDataSpecResponse.ProtoReflect.Descriptor instead.
func (*QueryGetDataSpecResponse) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{3}
}

func (x *QueryGetDataSpecResponse) GetSpec() *DataSpec {
	if x != nil {
		return x.Spec
	}
	return nil
}

// QueryDecodeQuerydataRequest is request type for the Query/DecodeQuerydata RPC method.
type QueryDecodeQuerydataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// query_data is the query data hex string to be decoded.
	QueryData []byte `protobuf:"bytes,1,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
}

func (x *QueryDecodeQuerydataRequest) Reset() {
	*x = QueryDecodeQuerydataRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryDecodeQuerydataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryDecodeQuerydataRequest) ProtoMessage() {}

func (x *QueryDecodeQuerydataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryDecodeQuerydataRequest.ProtoReflect.Descriptor instead.
func (*QueryDecodeQuerydataRequest) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{4}
}

func (x *QueryDecodeQuerydataRequest) GetQueryData() []byte {
	if x != nil {
		return x.QueryData
	}
	return nil
}

// QueryDecodeQuerydataResponse is response type for the Query/DecodeQuerydata RPC method.
type QueryDecodeQuerydataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// spec is the decoded json represention of the query data hex string.
	Spec string `protobuf:"bytes,1,opt,name=spec,proto3" json:"spec,omitempty"`
}

func (x *QueryDecodeQuerydataResponse) Reset() {
	*x = QueryDecodeQuerydataResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryDecodeQuerydataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryDecodeQuerydataResponse) ProtoMessage() {}

func (x *QueryDecodeQuerydataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryDecodeQuerydataResponse.ProtoReflect.Descriptor instead.
func (*QueryDecodeQuerydataResponse) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{5}
}

func (x *QueryDecodeQuerydataResponse) GetSpec() string {
	if x != nil {
		return x.Spec
	}
	return ""
}

// QueryGenerateQuerydataRequest is request type for the Query/GenerateQuerydata RPC method.
type QueryGenerateQuerydataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// querytype for which query_data is to be generated.
	Querytype string `protobuf:"bytes,1,opt,name=querytype,proto3" json:"querytype,omitempty"`
	// parameters for which query_data is to be generated.
	Parameters string `protobuf:"bytes,2,opt,name=parameters,proto3" json:"parameters,omitempty"`
}

func (x *QueryGenerateQuerydataRequest) Reset() {
	*x = QueryGenerateQuerydataRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryGenerateQuerydataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryGenerateQuerydataRequest) ProtoMessage() {}

func (x *QueryGenerateQuerydataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryGenerateQuerydataRequest.ProtoReflect.Descriptor instead.
func (*QueryGenerateQuerydataRequest) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{6}
}

func (x *QueryGenerateQuerydataRequest) GetQuerytype() string {
	if x != nil {
		return x.Querytype
	}
	return ""
}

func (x *QueryGenerateQuerydataRequest) GetParameters() string {
	if x != nil {
		return x.Parameters
	}
	return ""
}

// QueryGenerateQuerydataResponse is response type for the Query/GenerateQuerydata RPC method.
type QueryGenerateQuerydataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// query_data is the generated query_data hex string.
	QueryData []byte `protobuf:"bytes,1,opt,name=query_data,json=queryData,proto3" json:"query_data,omitempty"`
}

func (x *QueryGenerateQuerydataResponse) Reset() {
	*x = QueryGenerateQuerydataResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryGenerateQuerydataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryGenerateQuerydataResponse) ProtoMessage() {}

func (x *QueryGenerateQuerydataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryGenerateQuerydataResponse.ProtoReflect.Descriptor instead.
func (*QueryGenerateQuerydataResponse) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{7}
}

func (x *QueryGenerateQuerydataResponse) GetQueryData() []byte {
	if x != nil {
		return x.QueryData
	}
	return nil
}

// QueryDecodeValueRequest is request type for the Query/DecodeValue RPC method.
type QueryDecodeValueRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// queryType is the key to fetch a the corresponding data spec.
	QueryType string `protobuf:"bytes,1,opt,name=queryType,proto3" json:"queryType,omitempty"`
	// value is the value hex string to be decoded.
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *QueryDecodeValueRequest) Reset() {
	*x = QueryDecodeValueRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryDecodeValueRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryDecodeValueRequest) ProtoMessage() {}

func (x *QueryDecodeValueRequest) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryDecodeValueRequest.ProtoReflect.Descriptor instead.
func (*QueryDecodeValueRequest) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{8}
}

func (x *QueryDecodeValueRequest) GetQueryType() string {
	if x != nil {
		return x.QueryType
	}
	return ""
}

func (x *QueryDecodeValueRequest) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

// QueryDecodeValueResponse is response type for the Query/DecodeValue RPC method.
type QueryDecodeValueResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// decodedValue is the decoded value of the hex string.
	DecodedValue string `protobuf:"bytes,1,opt,name=decodedValue,proto3" json:"decodedValue,omitempty"`
}

func (x *QueryDecodeValueResponse) Reset() {
	*x = QueryDecodeValueResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_registry_query_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryDecodeValueResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryDecodeValueResponse) ProtoMessage() {}

func (x *QueryDecodeValueResponse) ProtoReflect() protoreflect.Message {
	mi := &file_layer_registry_query_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryDecodeValueResponse.ProtoReflect.Descriptor instead.
func (*QueryDecodeValueResponse) Descriptor() ([]byte, []int) {
	return file_layer_registry_query_proto_rawDescGZIP(), []int{9}
}

func (x *QueryDecodeValueResponse) GetDecodedValue() string {
	if x != nil {
		return x.DecodedValue
	}
	return ""
}

var File_layer_registry_query_proto protoreflect.FileDescriptor

var file_layer_registry_query_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x1a, 0x14, 0x67, 0x6f,
	0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61,
	0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2f, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x73, 0x70, 0x65, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1b, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x14, 0x0a,
	0x12, 0x51, 0x75, 0x65, 0x72, 0x79, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x22, 0x4b, 0x0a, 0x13, 0x51, 0x75, 0x65, 0x72, 0x79, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x34, 0x0a, 0x06, 0x70, 0x61,
	0x72, 0x61, 0x6d, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x73, 0x42, 0x04, 0xc8, 0xde, 0x1f, 0x00, 0x52, 0x06, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73,
	0x22, 0x38, 0x0a, 0x17, 0x51, 0x75, 0x65, 0x72, 0x79, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61,
	0x53, 0x70, 0x65, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x71, 0x75, 0x65, 0x72, 0x79, 0x54, 0x79, 0x70, 0x65, 0x22, 0x48, 0x0a, 0x18, 0x51, 0x75,
	0x65, 0x72, 0x79, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x53, 0x70, 0x65, 0x63, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2c, 0x0a, 0x04, 0x73, 0x70, 0x65, 0x63, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x53, 0x70, 0x65, 0x63, 0x52, 0x04,
	0x73, 0x70, 0x65, 0x63, 0x22, 0x3c, 0x0a, 0x1b, 0x51, 0x75, 0x65, 0x72, 0x79, 0x44, 0x65, 0x63,
	0x6f, 0x64, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x64, 0x61, 0x74,
	0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x71, 0x75, 0x65, 0x72, 0x79, 0x44, 0x61,
	0x74, 0x61, 0x22, 0x32, 0x0a, 0x1c, 0x51, 0x75, 0x65, 0x72, 0x79, 0x44, 0x65, 0x63, 0x6f, 0x64,
	0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x70, 0x65, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x73, 0x70, 0x65, 0x63, 0x22, 0x5d, 0x0a, 0x1d, 0x51, 0x75, 0x65, 0x72, 0x79, 0x47,
	0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x71, 0x75, 0x65, 0x72,
	0x79, 0x74, 0x79, 0x70, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74,
	0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x73, 0x22, 0x3f, 0x0a, 0x1e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x47, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x44, 0x61, 0x74, 0x61, 0x22, 0x4d, 0x0a, 0x17, 0x51, 0x75, 0x65, 0x72, 0x79, 0x44,
	0x65, 0x63, 0x6f, 0x64, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x1c, 0x0a, 0x09, 0x71, 0x75, 0x65, 0x72, 0x79, 0x54, 0x79, 0x70, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x71, 0x75, 0x65, 0x72, 0x79, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x3e, 0x0a, 0x18, 0x51, 0x75, 0x65, 0x72, 0x79, 0x44, 0x65,
	0x63, 0x6f, 0x64, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x22, 0x0a, 0x0c, 0x64, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x64, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x64, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x64,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x32, 0x8e, 0x06, 0x0a, 0x05, 0x51, 0x75, 0x65, 0x72, 0x79, 0x12,
	0x71, 0x0a, 0x06, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x12, 0x22, 0x2e, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79,
	0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x23, 0x2e,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51,
	0x75, 0x65, 0x72, 0x79, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12, 0x16, 0x2f, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x70, 0x61, 0x72, 0x61,
	0x6d, 0x73, 0x12, 0x94, 0x01, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x53, 0x70,
	0x65, 0x63, 0x12, 0x27, 0x2e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61,
	0x53, 0x70, 0x65, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28, 0x2e, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x53, 0x70, 0x65, 0x63, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x32, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x2c, 0x12, 0x2a, 0x2f,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x67,
	0x65, 0x74, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x73, 0x70, 0x65, 0x63, 0x2f, 0x7b, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x7d, 0x12, 0xa3, 0x01, 0x0a, 0x0f, 0x44, 0x65,
	0x63, 0x6f, 0x64, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61, 0x12, 0x2b, 0x2e,
	0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51,
	0x75, 0x65, 0x72, 0x79, 0x44, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64,
	0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2c, 0x2e, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72,
	0x79, 0x44, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x35, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x2f,
	0x12, 0x2d, 0x2f, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2f, 0x64, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61,
	0x74, 0x61, 0x2f, 0x7b, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x7d, 0x12,
	0xb7, 0x01, 0x0a, 0x11, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72,
	0x79, 0x64, 0x61, 0x74, 0x61, 0x12, 0x2d, 0x2e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x47, 0x65, 0x6e, 0x65,
	0x72, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x47, 0x65, 0x6e, 0x65, 0x72,
	0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x43, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x3d, 0x12, 0x3b, 0x2f, 0x6c,
	0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x64, 0x61, 0x74, 0x61,
	0x2f, 0x7b, 0x71, 0x75, 0x65, 0x72, 0x79, 0x74, 0x79, 0x70, 0x65, 0x7d, 0x2f, 0x7b, 0x70, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x7d, 0x12, 0x9a, 0x01, 0x0a, 0x0b, 0x44, 0x65,
	0x63, 0x6f, 0x64, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x27, 0x2e, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79,
	0x44, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x28, 0x2e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x44, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x38, 0x82, 0xd3,
	0xe4, 0x93, 0x02, 0x32, 0x12, 0x30, 0x2f, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x64, 0x65, 0x63, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x2f, 0x7b, 0x71, 0x75, 0x65, 0x72, 0x79, 0x54, 0x79, 0x70, 0x65, 0x7d, 0x2f, 0x7b,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x7d, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x65, 0x6c, 0x6c, 0x6f, 0x72, 0x2d, 0x69, 0x6f, 0x2f, 0x6c,
	0x61, 0x79, 0x65, 0x72, 0x2f, 0x78, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f,
	0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_layer_registry_query_proto_rawDescOnce sync.Once
	file_layer_registry_query_proto_rawDescData = file_layer_registry_query_proto_rawDesc
)

func file_layer_registry_query_proto_rawDescGZIP() []byte {
	file_layer_registry_query_proto_rawDescOnce.Do(func() {
		file_layer_registry_query_proto_rawDescData = protoimpl.X.CompressGZIP(file_layer_registry_query_proto_rawDescData)
	})
	return file_layer_registry_query_proto_rawDescData
}

var file_layer_registry_query_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_layer_registry_query_proto_goTypes = []interface{}{
	(*QueryParamsRequest)(nil),             // 0: layer.registry.QueryParamsRequest
	(*QueryParamsResponse)(nil),            // 1: layer.registry.QueryParamsResponse
	(*QueryGetDataSpecRequest)(nil),        // 2: layer.registry.QueryGetDataSpecRequest
	(*QueryGetDataSpecResponse)(nil),       // 3: layer.registry.QueryGetDataSpecResponse
	(*QueryDecodeQuerydataRequest)(nil),    // 4: layer.registry.QueryDecodeQuerydataRequest
	(*QueryDecodeQuerydataResponse)(nil),   // 5: layer.registry.QueryDecodeQuerydataResponse
	(*QueryGenerateQuerydataRequest)(nil),  // 6: layer.registry.QueryGenerateQuerydataRequest
	(*QueryGenerateQuerydataResponse)(nil), // 7: layer.registry.QueryGenerateQuerydataResponse
	(*QueryDecodeValueRequest)(nil),        // 8: layer.registry.QueryDecodeValueRequest
	(*QueryDecodeValueResponse)(nil),       // 9: layer.registry.QueryDecodeValueResponse
	(*Params)(nil),                         // 10: layer.registry.Params
	(*DataSpec)(nil),                       // 11: layer.registry.DataSpec
}
var file_layer_registry_query_proto_depIdxs = []int32{
	10, // 0: layer.registry.QueryParamsResponse.params:type_name -> layer.registry.Params
	11, // 1: layer.registry.QueryGetDataSpecResponse.spec:type_name -> layer.registry.DataSpec
	0,  // 2: layer.registry.Query.Params:input_type -> layer.registry.QueryParamsRequest
	2,  // 3: layer.registry.Query.GetDataSpec:input_type -> layer.registry.QueryGetDataSpecRequest
	4,  // 4: layer.registry.Query.DecodeQuerydata:input_type -> layer.registry.QueryDecodeQuerydataRequest
	6,  // 5: layer.registry.Query.GenerateQuerydata:input_type -> layer.registry.QueryGenerateQuerydataRequest
	8,  // 6: layer.registry.Query.DecodeValue:input_type -> layer.registry.QueryDecodeValueRequest
	1,  // 7: layer.registry.Query.Params:output_type -> layer.registry.QueryParamsResponse
	3,  // 8: layer.registry.Query.GetDataSpec:output_type -> layer.registry.QueryGetDataSpecResponse
	5,  // 9: layer.registry.Query.DecodeQuerydata:output_type -> layer.registry.QueryDecodeQuerydataResponse
	7,  // 10: layer.registry.Query.GenerateQuerydata:output_type -> layer.registry.QueryGenerateQuerydataResponse
	9,  // 11: layer.registry.Query.DecodeValue:output_type -> layer.registry.QueryDecodeValueResponse
	7,  // [7:12] is the sub-list for method output_type
	2,  // [2:7] is the sub-list for method input_type
	2,  // [2:2] is the sub-list for extension type_name
	2,  // [2:2] is the sub-list for extension extendee
	0,  // [0:2] is the sub-list for field type_name
}

func init() { file_layer_registry_query_proto_init() }
func file_layer_registry_query_proto_init() {
	if File_layer_registry_query_proto != nil {
		return
	}
	file_layer_registry_data_spec_proto_init()
	file_layer_registry_params_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_layer_registry_query_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryParamsRequest); i {
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
		file_layer_registry_query_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryParamsResponse); i {
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
		file_layer_registry_query_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryGetDataSpecRequest); i {
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
		file_layer_registry_query_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryGetDataSpecResponse); i {
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
		file_layer_registry_query_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryDecodeQuerydataRequest); i {
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
		file_layer_registry_query_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryDecodeQuerydataResponse); i {
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
		file_layer_registry_query_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryGenerateQuerydataRequest); i {
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
		file_layer_registry_query_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryGenerateQuerydataResponse); i {
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
		file_layer_registry_query_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryDecodeValueRequest); i {
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
		file_layer_registry_query_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryDecodeValueResponse); i {
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
			RawDescriptor: file_layer_registry_query_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_layer_registry_query_proto_goTypes,
		DependencyIndexes: file_layer_registry_query_proto_depIdxs,
		MessageInfos:      file_layer_registry_query_proto_msgTypes,
	}.Build()
	File_layer_registry_query_proto = out.File
	file_layer_registry_query_proto_rawDesc = nil
	file_layer_registry_query_proto_goTypes = nil
	file_layer_registry_query_proto_depIdxs = nil
}
