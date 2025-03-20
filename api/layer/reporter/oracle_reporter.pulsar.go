// Code generated by protoc-gen-go-pulsar. DO NOT EDIT.
package reporter

import (
	_ "cosmossdk.io/api/amino"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	runtime "github.com/cosmos/cosmos-proto/runtime"
	_ "github.com/cosmos/gogoproto/gogoproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	reflect "reflect"
	sync "sync"
)

var (
	md_OracleReporter                     protoreflect.MessageDescriptor
	fd_OracleReporter_min_tokens_required protoreflect.FieldDescriptor
	fd_OracleReporter_commission_rate     protoreflect.FieldDescriptor
	fd_OracleReporter_jailed              protoreflect.FieldDescriptor
	fd_OracleReporter_jailed_until        protoreflect.FieldDescriptor
	fd_OracleReporter_moniker             protoreflect.FieldDescriptor
	fd_OracleReporter_last_updated        protoreflect.FieldDescriptor
)

func init() {
	file_layer_reporter_oracle_reporter_proto_init()
	md_OracleReporter = File_layer_reporter_oracle_reporter_proto.Messages().ByName("OracleReporter")
	fd_OracleReporter_min_tokens_required = md_OracleReporter.Fields().ByName("min_tokens_required")
	fd_OracleReporter_commission_rate = md_OracleReporter.Fields().ByName("commission_rate")
	fd_OracleReporter_jailed = md_OracleReporter.Fields().ByName("jailed")
	fd_OracleReporter_jailed_until = md_OracleReporter.Fields().ByName("jailed_until")
	fd_OracleReporter_moniker = md_OracleReporter.Fields().ByName("moniker")
	fd_OracleReporter_last_updated = md_OracleReporter.Fields().ByName("last_updated")
}

var _ protoreflect.Message = (*fastReflection_OracleReporter)(nil)

type fastReflection_OracleReporter OracleReporter

func (x *OracleReporter) ProtoReflect() protoreflect.Message {
	return (*fastReflection_OracleReporter)(x)
}

func (x *OracleReporter) slowProtoReflect() protoreflect.Message {
	mi := &file_layer_reporter_oracle_reporter_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var _fastReflection_OracleReporter_messageType fastReflection_OracleReporter_messageType
var _ protoreflect.MessageType = fastReflection_OracleReporter_messageType{}

type fastReflection_OracleReporter_messageType struct{}

func (x fastReflection_OracleReporter_messageType) Zero() protoreflect.Message {
	return (*fastReflection_OracleReporter)(nil)
}
func (x fastReflection_OracleReporter_messageType) New() protoreflect.Message {
	return new(fastReflection_OracleReporter)
}
func (x fastReflection_OracleReporter_messageType) Descriptor() protoreflect.MessageDescriptor {
	return md_OracleReporter
}

// Descriptor returns message descriptor, which contains only the protobuf
// type information for the message.
func (x *fastReflection_OracleReporter) Descriptor() protoreflect.MessageDescriptor {
	return md_OracleReporter
}

// Type returns the message type, which encapsulates both Go and protobuf
// type information. If the Go type information is not needed,
// it is recommended that the message descriptor be used instead.
func (x *fastReflection_OracleReporter) Type() protoreflect.MessageType {
	return _fastReflection_OracleReporter_messageType
}

// New returns a newly allocated and mutable empty message.
func (x *fastReflection_OracleReporter) New() protoreflect.Message {
	return new(fastReflection_OracleReporter)
}

// Interface unwraps the message reflection interface and
// returns the underlying ProtoMessage interface.
func (x *fastReflection_OracleReporter) Interface() protoreflect.ProtoMessage {
	return (*OracleReporter)(x)
}

// Range iterates over every populated field in an undefined order,
// calling f for each field descriptor and value encountered.
// Range returns immediately if f returns false.
// While iterating, mutating operations may only be performed
// on the current field descriptor.
func (x *fastReflection_OracleReporter) Range(f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if x.MinTokensRequired != "" {
		value := protoreflect.ValueOfString(x.MinTokensRequired)
		if !f(fd_OracleReporter_min_tokens_required, value) {
			return
		}
	}
	if x.CommissionRate != "" {
		value := protoreflect.ValueOfString(x.CommissionRate)
		if !f(fd_OracleReporter_commission_rate, value) {
			return
		}
	}
	if x.Jailed != false {
		value := protoreflect.ValueOfBool(x.Jailed)
		if !f(fd_OracleReporter_jailed, value) {
			return
		}
	}
	if x.JailedUntil != nil {
		value := protoreflect.ValueOfMessage(x.JailedUntil.ProtoReflect())
		if !f(fd_OracleReporter_jailed_until, value) {
			return
		}
	}
	if x.Moniker != "" {
		value := protoreflect.ValueOfString(x.Moniker)
		if !f(fd_OracleReporter_moniker, value) {
			return
		}
	}
	if x.LastUpdated != nil {
		value := protoreflect.ValueOfMessage(x.LastUpdated.ProtoReflect())
		if !f(fd_OracleReporter_last_updated, value) {
			return
		}
	}
}

// Has reports whether a field is populated.
//
// Some fields have the property of nullability where it is possible to
// distinguish between the default value of a field and whether the field
// was explicitly populated with the default value. Singular message fields,
// member fields of a oneof, and proto2 scalar fields are nullable. Such
// fields are populated only if explicitly set.
//
// In other cases (aside from the nullable cases above),
// a proto3 scalar field is populated if it contains a non-zero value, and
// a repeated field is populated if it is non-empty.
func (x *fastReflection_OracleReporter) Has(fd protoreflect.FieldDescriptor) bool {
	switch fd.FullName() {
	case "layer.reporter.OracleReporter.min_tokens_required":
		return x.MinTokensRequired != ""
	case "layer.reporter.OracleReporter.commission_rate":
		return x.CommissionRate != ""
	case "layer.reporter.OracleReporter.jailed":
		return x.Jailed != false
	case "layer.reporter.OracleReporter.jailed_until":
		return x.JailedUntil != nil
	case "layer.reporter.OracleReporter.moniker":
		return x.Moniker != ""
	case "layer.reporter.OracleReporter.last_updated":
		return x.LastUpdated != nil
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: layer.reporter.OracleReporter"))
		}
		panic(fmt.Errorf("message layer.reporter.OracleReporter does not contain field %s", fd.FullName()))
	}
}

// Clear clears the field such that a subsequent Has call reports false.
//
// Clearing an extension field clears both the extension type and value
// associated with the given field number.
//
// Clear is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_OracleReporter) Clear(fd protoreflect.FieldDescriptor) {
	switch fd.FullName() {
	case "layer.reporter.OracleReporter.min_tokens_required":
		x.MinTokensRequired = ""
	case "layer.reporter.OracleReporter.commission_rate":
		x.CommissionRate = ""
	case "layer.reporter.OracleReporter.jailed":
		x.Jailed = false
	case "layer.reporter.OracleReporter.jailed_until":
		x.JailedUntil = nil
	case "layer.reporter.OracleReporter.moniker":
		x.Moniker = ""
	case "layer.reporter.OracleReporter.last_updated":
		x.LastUpdated = nil
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: layer.reporter.OracleReporter"))
		}
		panic(fmt.Errorf("message layer.reporter.OracleReporter does not contain field %s", fd.FullName()))
	}
}

// Get retrieves the value for a field.
//
// For unpopulated scalars, it returns the default value, where
// the default value of a bytes scalar is guaranteed to be a copy.
// For unpopulated composite types, it returns an empty, read-only view
// of the value; to obtain a mutable reference, use Mutable.
func (x *fastReflection_OracleReporter) Get(descriptor protoreflect.FieldDescriptor) protoreflect.Value {
	switch descriptor.FullName() {
	case "layer.reporter.OracleReporter.min_tokens_required":
		value := x.MinTokensRequired
		return protoreflect.ValueOfString(value)
	case "layer.reporter.OracleReporter.commission_rate":
		value := x.CommissionRate
		return protoreflect.ValueOfString(value)
	case "layer.reporter.OracleReporter.jailed":
		value := x.Jailed
		return protoreflect.ValueOfBool(value)
	case "layer.reporter.OracleReporter.jailed_until":
		value := x.JailedUntil
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	case "layer.reporter.OracleReporter.moniker":
		value := x.Moniker
		return protoreflect.ValueOfString(value)
	case "layer.reporter.OracleReporter.last_updated":
		value := x.LastUpdated
		return protoreflect.ValueOfMessage(value.ProtoReflect())
	default:
		if descriptor.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: layer.reporter.OracleReporter"))
		}
		panic(fmt.Errorf("message layer.reporter.OracleReporter does not contain field %s", descriptor.FullName()))
	}
}

// Set stores the value for a field.
//
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType.
// When setting a composite type, it is unspecified whether the stored value
// aliases the source's memory in any way. If the composite value is an
// empty, read-only value, then it panics.
//
// Set is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_OracleReporter) Set(fd protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch fd.FullName() {
	case "layer.reporter.OracleReporter.min_tokens_required":
		x.MinTokensRequired = value.Interface().(string)
	case "layer.reporter.OracleReporter.commission_rate":
		x.CommissionRate = value.Interface().(string)
	case "layer.reporter.OracleReporter.jailed":
		x.Jailed = value.Bool()
	case "layer.reporter.OracleReporter.jailed_until":
		x.JailedUntil = value.Message().Interface().(*timestamppb.Timestamp)
	case "layer.reporter.OracleReporter.moniker":
		x.Moniker = value.Interface().(string)
	case "layer.reporter.OracleReporter.last_updated":
		x.LastUpdated = value.Message().Interface().(*timestamppb.Timestamp)
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: layer.reporter.OracleReporter"))
		}
		panic(fmt.Errorf("message layer.reporter.OracleReporter does not contain field %s", fd.FullName()))
	}
}

// Mutable returns a mutable reference to a composite type.
//
// If the field is unpopulated, it may allocate a composite value.
// For a field belonging to a oneof, it implicitly clears any other field
// that may be currently set within the same oneof.
// For extension fields, it implicitly stores the provided ExtensionType
// if not already stored.
// It panics if the field does not contain a composite type.
//
// Mutable is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_OracleReporter) Mutable(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "layer.reporter.OracleReporter.jailed_until":
		if x.JailedUntil == nil {
			x.JailedUntil = new(timestamppb.Timestamp)
		}
		return protoreflect.ValueOfMessage(x.JailedUntil.ProtoReflect())
	case "layer.reporter.OracleReporter.last_updated":
		if x.LastUpdated == nil {
			x.LastUpdated = new(timestamppb.Timestamp)
		}
		return protoreflect.ValueOfMessage(x.LastUpdated.ProtoReflect())
	case "layer.reporter.OracleReporter.min_tokens_required":
		panic(fmt.Errorf("field min_tokens_required of message layer.reporter.OracleReporter is not mutable"))
	case "layer.reporter.OracleReporter.commission_rate":
		panic(fmt.Errorf("field commission_rate of message layer.reporter.OracleReporter is not mutable"))
	case "layer.reporter.OracleReporter.jailed":
		panic(fmt.Errorf("field jailed of message layer.reporter.OracleReporter is not mutable"))
	case "layer.reporter.OracleReporter.moniker":
		panic(fmt.Errorf("field moniker of message layer.reporter.OracleReporter is not mutable"))
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: layer.reporter.OracleReporter"))
		}
		panic(fmt.Errorf("message layer.reporter.OracleReporter does not contain field %s", fd.FullName()))
	}
}

// NewField returns a new value that is assignable to the field
// for the given descriptor. For scalars, this returns the default value.
// For lists, maps, and messages, this returns a new, empty, mutable value.
func (x *fastReflection_OracleReporter) NewField(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.FullName() {
	case "layer.reporter.OracleReporter.min_tokens_required":
		return protoreflect.ValueOfString("")
	case "layer.reporter.OracleReporter.commission_rate":
		return protoreflect.ValueOfString("")
	case "layer.reporter.OracleReporter.jailed":
		return protoreflect.ValueOfBool(false)
	case "layer.reporter.OracleReporter.jailed_until":
		m := new(timestamppb.Timestamp)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	case "layer.reporter.OracleReporter.moniker":
		return protoreflect.ValueOfString("")
	case "layer.reporter.OracleReporter.last_updated":
		m := new(timestamppb.Timestamp)
		return protoreflect.ValueOfMessage(m.ProtoReflect())
	default:
		if fd.IsExtension() {
			panic(fmt.Errorf("proto3 declared messages do not support extensions: layer.reporter.OracleReporter"))
		}
		panic(fmt.Errorf("message layer.reporter.OracleReporter does not contain field %s", fd.FullName()))
	}
}

// WhichOneof reports which field within the oneof is populated,
// returning nil if none are populated.
// It panics if the oneof descriptor does not belong to this message.
func (x *fastReflection_OracleReporter) WhichOneof(d protoreflect.OneofDescriptor) protoreflect.FieldDescriptor {
	switch d.FullName() {
	default:
		panic(fmt.Errorf("%s is not a oneof field in layer.reporter.OracleReporter", d.FullName()))
	}
	panic("unreachable")
}

// GetUnknown retrieves the entire list of unknown fields.
// The caller may only mutate the contents of the RawFields
// if the mutated bytes are stored back into the message with SetUnknown.
func (x *fastReflection_OracleReporter) GetUnknown() protoreflect.RawFields {
	return x.unknownFields
}

// SetUnknown stores an entire list of unknown fields.
// The raw fields must be syntactically valid according to the wire format.
// An implementation may panic if this is not the case.
// Once stored, the caller must not mutate the content of the RawFields.
// An empty RawFields may be passed to clear the fields.
//
// SetUnknown is a mutating operation and unsafe for concurrent use.
func (x *fastReflection_OracleReporter) SetUnknown(fields protoreflect.RawFields) {
	x.unknownFields = fields
}

// IsValid reports whether the message is valid.
//
// An invalid message is an empty, read-only value.
//
// An invalid message often corresponds to a nil pointer of the concrete
// message type, but the details are implementation dependent.
// Validity is not part of the protobuf data model, and may not
// be preserved in marshaling or other operations.
func (x *fastReflection_OracleReporter) IsValid() bool {
	return x != nil
}

// ProtoMethods returns optional fastReflectionFeature-path implementations of various operations.
// This method may return nil.
//
// The returned methods type is identical to
// "google.golang.org/protobuf/runtime/protoiface".Methods.
// Consult the protoiface package documentation for details.
func (x *fastReflection_OracleReporter) ProtoMethods() *protoiface.Methods {
	size := func(input protoiface.SizeInput) protoiface.SizeOutput {
		x := input.Message.Interface().(*OracleReporter)
		if x == nil {
			return protoiface.SizeOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Size:              0,
			}
		}
		options := runtime.SizeInputToOptions(input)
		_ = options
		var n int
		var l int
		_ = l
		l = len(x.MinTokensRequired)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.CommissionRate)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.Jailed {
			n += 2
		}
		if x.JailedUntil != nil {
			l = options.Size(x.JailedUntil)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		l = len(x.Moniker)
		if l > 0 {
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.LastUpdated != nil {
			l = options.Size(x.LastUpdated)
			n += 1 + l + runtime.Sov(uint64(l))
		}
		if x.unknownFields != nil {
			n += len(x.unknownFields)
		}
		return protoiface.SizeOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Size:              n,
		}
	}

	marshal := func(input protoiface.MarshalInput) (protoiface.MarshalOutput, error) {
		x := input.Message.Interface().(*OracleReporter)
		if x == nil {
			return protoiface.MarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Buf:               input.Buf,
			}, nil
		}
		options := runtime.MarshalInputToOptions(input)
		_ = options
		size := options.Size(x)
		dAtA := make([]byte, size)
		i := len(dAtA)
		_ = i
		var l int
		_ = l
		if x.unknownFields != nil {
			i -= len(x.unknownFields)
			copy(dAtA[i:], x.unknownFields)
		}
		if x.LastUpdated != nil {
			encoded, err := options.Marshal(x.LastUpdated)
			if err != nil {
				return protoiface.MarshalOutput{
					NoUnkeyedLiterals: input.NoUnkeyedLiterals,
					Buf:               input.Buf,
				}, err
			}
			i -= len(encoded)
			copy(dAtA[i:], encoded)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
			i--
			dAtA[i] = 0x32
		}
		if len(x.Moniker) > 0 {
			i -= len(x.Moniker)
			copy(dAtA[i:], x.Moniker)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.Moniker)))
			i--
			dAtA[i] = 0x2a
		}
		if x.JailedUntil != nil {
			encoded, err := options.Marshal(x.JailedUntil)
			if err != nil {
				return protoiface.MarshalOutput{
					NoUnkeyedLiterals: input.NoUnkeyedLiterals,
					Buf:               input.Buf,
				}, err
			}
			i -= len(encoded)
			copy(dAtA[i:], encoded)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(encoded)))
			i--
			dAtA[i] = 0x22
		}
		if x.Jailed {
			i--
			if x.Jailed {
				dAtA[i] = 1
			} else {
				dAtA[i] = 0
			}
			i--
			dAtA[i] = 0x18
		}
		if len(x.CommissionRate) > 0 {
			i -= len(x.CommissionRate)
			copy(dAtA[i:], x.CommissionRate)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.CommissionRate)))
			i--
			dAtA[i] = 0x12
		}
		if len(x.MinTokensRequired) > 0 {
			i -= len(x.MinTokensRequired)
			copy(dAtA[i:], x.MinTokensRequired)
			i = runtime.EncodeVarint(dAtA, i, uint64(len(x.MinTokensRequired)))
			i--
			dAtA[i] = 0xa
		}
		if input.Buf != nil {
			input.Buf = append(input.Buf, dAtA...)
		} else {
			input.Buf = dAtA
		}
		return protoiface.MarshalOutput{
			NoUnkeyedLiterals: input.NoUnkeyedLiterals,
			Buf:               input.Buf,
		}, nil
	}
	unmarshal := func(input protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
		x := input.Message.Interface().(*OracleReporter)
		if x == nil {
			return protoiface.UnmarshalOutput{
				NoUnkeyedLiterals: input.NoUnkeyedLiterals,
				Flags:             input.Flags,
			}, nil
		}
		options := runtime.UnmarshalInputToOptions(input)
		_ = options
		dAtA := input.Buf
		l := len(dAtA)
		iNdEx := 0
		for iNdEx < l {
			preIndex := iNdEx
			var wire uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
				}
				if iNdEx >= l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				wire |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			fieldNum := int32(wire >> 3)
			wireType := int(wire & 0x7)
			if wireType == 4 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: OracleReporter: wiretype end group for non-group")
			}
			if fieldNum <= 0 {
				return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: OracleReporter: illegal tag %d (wire type %d)", fieldNum, wire)
			}
			switch fieldNum {
			case 1:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field MinTokensRequired", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.MinTokensRequired = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 2:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field CommissionRate", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.CommissionRate = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 3:
				if wireType != 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Jailed", wireType)
				}
				var v int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				x.Jailed = bool(v != 0)
			case 4:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field JailedUntil", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if x.JailedUntil == nil {
					x.JailedUntil = &timestamppb.Timestamp{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.JailedUntil); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			case 5:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field Moniker", wireType)
				}
				var stringLen uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLen |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLen := int(stringLen)
				if intStringLen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + intStringLen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				x.Moniker = string(dAtA[iNdEx:postIndex])
				iNdEx = postIndex
			case 6:
				if wireType != 2 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, fmt.Errorf("proto: wrong wireType = %d for field LastUpdated", wireType)
				}
				var msglen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrIntOverflow
					}
					if iNdEx >= l {
						return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					msglen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if msglen < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				postIndex := iNdEx + msglen
				if postIndex < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if postIndex > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if x.LastUpdated == nil {
					x.LastUpdated = &timestamppb.Timestamp{}
				}
				if err := options.Unmarshal(dAtA[iNdEx:postIndex], x.LastUpdated); err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				iNdEx = postIndex
			default:
				iNdEx = preIndex
				skippy, err := runtime.Skip(dAtA[iNdEx:])
				if err != nil {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, err
				}
				if (skippy < 0) || (iNdEx+skippy) < 0 {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, runtime.ErrInvalidLength
				}
				if (iNdEx + skippy) > l {
					return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
				}
				if !options.DiscardUnknown {
					x.unknownFields = append(x.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
				}
				iNdEx += skippy
			}
		}

		if iNdEx > l {
			return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, io.ErrUnexpectedEOF
		}
		return protoiface.UnmarshalOutput{NoUnkeyedLiterals: input.NoUnkeyedLiterals, Flags: input.Flags}, nil
	}
	return &protoiface.Methods{
		NoUnkeyedLiterals: struct{}{},
		Flags:             protoiface.SupportMarshalDeterministic | protoiface.SupportUnmarshalDiscardUnknown,
		Size:              size,
		Marshal:           marshal,
		Unmarshal:         unmarshal,
		Merge:             nil,
		CheckInitialized:  nil,
	}
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.0
// 	protoc        (unknown)
// source: layer/reporter/oracle_reporter.proto

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// OracleReporter is the struct that holds the data for a reporter
type OracleReporter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// min_tokens_required to select this reporter
	MinTokensRequired string `protobuf:"bytes,1,opt,name=min_tokens_required,json=minTokensRequired,proto3" json:"min_tokens_required,omitempty"`
	// commission for the reporter
	CommissionRate string `protobuf:"bytes,2,opt,name=commission_rate,json=commissionRate,proto3" json:"commission_rate,omitempty"`
	// jailed is a bool whether the reporter is jailed or not
	Jailed bool `protobuf:"varint,3,opt,name=jailed,proto3" json:"jailed,omitempty"`
	// jailed_until is the time the reporter is jailed until
	JailedUntil *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=jailed_until,json=jailedUntil,proto3" json:"jailed_until,omitempty"`
	// moniker is the moniker of the reporter
	Moniker string `protobuf:"bytes,5,opt,name=moniker,proto3" json:"moniker,omitempty"`
	// Time that the reporter was last updated
	LastUpdated *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=last_updated,json=lastUpdated,proto3" json:"last_updated,omitempty"`
}

func (x *OracleReporter) Reset() {
	*x = OracleReporter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_layer_reporter_oracle_reporter_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OracleReporter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OracleReporter) ProtoMessage() {}

// Deprecated: Use OracleReporter.ProtoReflect.Descriptor instead.
func (*OracleReporter) Descriptor() ([]byte, []int) {
	return file_layer_reporter_oracle_reporter_proto_rawDescGZIP(), []int{0}
}

func (x *OracleReporter) GetMinTokensRequired() string {
	if x != nil {
		return x.MinTokensRequired
	}
	return ""
}

func (x *OracleReporter) GetCommissionRate() string {
	if x != nil {
		return x.CommissionRate
	}
	return ""
}

func (x *OracleReporter) GetJailed() bool {
	if x != nil {
		return x.Jailed
	}
	return false
}

func (x *OracleReporter) GetJailedUntil() *timestamppb.Timestamp {
	if x != nil {
		return x.JailedUntil
	}
	return nil
}

func (x *OracleReporter) GetMoniker() string {
	if x != nil {
		return x.Moniker
	}
	return ""
}

func (x *OracleReporter) GetLastUpdated() *timestamppb.Timestamp {
	if x != nil {
		return x.LastUpdated
	}
	return nil
}

var File_layer_reporter_oracle_reporter_proto protoreflect.FileDescriptor

var file_layer_reporter_oracle_reporter_proto_rawDesc = []byte{
	0x0a, 0x24, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72,
	0x2f, 0x6f, 0x72, 0x61, 0x63, 0x6c, 0x65, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2e, 0x72, 0x65,
	0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x1a, 0x11, 0x61, 0x6d, 0x69, 0x6e, 0x6f, 0x2f, 0x61, 0x6d,
	0x69, 0x6e, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x63, 0x6f, 0x73, 0x6d, 0x6f,
	0x73, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x14, 0x67, 0x6f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x67, 0x6f, 0x67, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x97, 0x03, 0x0a, 0x0e,
	0x4f, 0x72, 0x61, 0x63, 0x6c, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x12, 0x5b,
	0x0a, 0x13, 0x6d, 0x69, 0x6e, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x5f, 0x72, 0x65, 0x71,
	0x75, 0x69, 0x72, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x2b, 0xc8, 0xde, 0x1f,
	0x00, 0xda, 0xde, 0x1f, 0x15, 0x63, 0x6f, 0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69,
	0x6f, 0x2f, 0x6d, 0x61, 0x74, 0x68, 0x2e, 0x49, 0x6e, 0x74, 0xd2, 0xb4, 0x2d, 0x0a, 0x63, 0x6f,
	0x73, 0x6d, 0x6f, 0x73, 0x2e, 0x49, 0x6e, 0x74, 0x52, 0x11, 0x6d, 0x69, 0x6e, 0x54, 0x6f, 0x6b,
	0x65, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x64, 0x12, 0x5a, 0x0a, 0x0f, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x72, 0x61, 0x74, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x31, 0xc8, 0xde, 0x1f, 0x00, 0xda, 0xde, 0x1f, 0x1b, 0x63, 0x6f,
	0x73, 0x6d, 0x6f, 0x73, 0x73, 0x64, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x6d, 0x61, 0x74, 0x68, 0x2e,
	0x4c, 0x65, 0x67, 0x61, 0x63, 0x79, 0x44, 0x65, 0x63, 0xd2, 0xb4, 0x2d, 0x0a, 0x63, 0x6f, 0x73,
	0x6d, 0x6f, 0x73, 0x2e, 0x44, 0x65, 0x63, 0x52, 0x0e, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x73, 0x73,
	0x69, 0x6f, 0x6e, 0x52, 0x61, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6a, 0x61, 0x69, 0x6c, 0x65,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x6a, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x12,
	0x4c, 0x0a, 0x0c, 0x6a, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x5f, 0x75, 0x6e, 0x74, 0x69, 0x6c, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x42, 0x0d, 0xc8, 0xde, 0x1f, 0x00, 0x90, 0xdf, 0x1f, 0x01, 0xa8, 0xe7, 0xb0, 0x2a, 0x01,
	0x52, 0x0b, 0x6a, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x55, 0x6e, 0x74, 0x69, 0x6c, 0x12, 0x18, 0x0a,
	0x07, 0x6d, 0x6f, 0x6e, 0x69, 0x6b, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x6d, 0x6f, 0x6e, 0x69, 0x6b, 0x65, 0x72, 0x12, 0x4c, 0x0a, 0x0c, 0x6c, 0x61, 0x73, 0x74, 0x5f,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0x0d, 0xc8, 0xde, 0x1f, 0x00, 0x90,
	0xdf, 0x1f, 0x01, 0xa8, 0xe7, 0xb0, 0x2a, 0x01, 0x52, 0x0b, 0x6c, 0x61, 0x73, 0x74, 0x55, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x64, 0x42, 0xb1, 0x01, 0x0a, 0x12, 0x63, 0x6f, 0x6d, 0x2e, 0x6c, 0x61,
	0x79, 0x65, 0x72, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x42, 0x13, 0x4f, 0x72,
	0x61, 0x63, 0x6c, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x50, 0x01, 0x5a, 0x2d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x74, 0x65, 0x6c, 0x6c, 0x6f, 0x72, 0x2d, 0x69, 0x6f, 0x2f, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f,
	0x61, 0x70, 0x69, 0x2f, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74,
	0x65, 0x72, 0xa2, 0x02, 0x03, 0x4c, 0x52, 0x58, 0xaa, 0x02, 0x0e, 0x4c, 0x61, 0x79, 0x65, 0x72,
	0x2e, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0xca, 0x02, 0x0e, 0x4c, 0x61, 0x79, 0x65,
	0x72, 0x5c, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0xe2, 0x02, 0x1a, 0x4c, 0x61, 0x79,
	0x65, 0x72, 0x5c, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0f, 0x4c, 0x61, 0x79, 0x65, 0x72, 0x3a,
	0x3a, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_layer_reporter_oracle_reporter_proto_rawDescOnce sync.Once
	file_layer_reporter_oracle_reporter_proto_rawDescData = file_layer_reporter_oracle_reporter_proto_rawDesc
)

func file_layer_reporter_oracle_reporter_proto_rawDescGZIP() []byte {
	file_layer_reporter_oracle_reporter_proto_rawDescOnce.Do(func() {
		file_layer_reporter_oracle_reporter_proto_rawDescData = protoimpl.X.CompressGZIP(file_layer_reporter_oracle_reporter_proto_rawDescData)
	})
	return file_layer_reporter_oracle_reporter_proto_rawDescData
}

var file_layer_reporter_oracle_reporter_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_layer_reporter_oracle_reporter_proto_goTypes = []interface{}{
	(*OracleReporter)(nil),        // 0: layer.reporter.OracleReporter
	(*timestamppb.Timestamp)(nil), // 1: google.protobuf.Timestamp
}
var file_layer_reporter_oracle_reporter_proto_depIdxs = []int32{
	1, // 0: layer.reporter.OracleReporter.jailed_until:type_name -> google.protobuf.Timestamp
	1, // 1: layer.reporter.OracleReporter.last_updated:type_name -> google.protobuf.Timestamp
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_layer_reporter_oracle_reporter_proto_init() }
func file_layer_reporter_oracle_reporter_proto_init() {
	if File_layer_reporter_oracle_reporter_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_layer_reporter_oracle_reporter_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OracleReporter); i {
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
			RawDescriptor: file_layer_reporter_oracle_reporter_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_layer_reporter_oracle_reporter_proto_goTypes,
		DependencyIndexes: file_layer_reporter_oracle_reporter_proto_depIdxs,
		MessageInfos:      file_layer_reporter_oracle_reporter_proto_msgTypes,
	}.Build()
	File_layer_reporter_oracle_reporter_proto = out.File
	file_layer_reporter_oracle_reporter_proto_rawDesc = nil
	file_layer_reporter_oracle_reporter_proto_goTypes = nil
	file_layer_reporter_oracle_reporter_proto_depIdxs = nil
}
