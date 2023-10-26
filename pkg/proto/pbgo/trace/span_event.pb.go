// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.9
// source: datadog/trace/span_event.proto

package trace

import (
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

type SpanEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// time is the number of nanoseconds between the Unix epoch and this event.
	// @gotags: json:"time" msg:"time"
	Time int64 `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	// name is this event's name.
	// @gotags: json:"name" msg:"name"
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// attributes is a mapping from attribute name to value for string-valued tags.
	// @gotags: json:"attributes" msg:"attributes"
	Attributes map[string]*AttributeValue `protobuf:"bytes,3,rep,name=attributes,proto3" json:"attributes,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// The number of attributes that could not be captured into the `attributes` field.
	// @gotags: json:"dropped_attributes_count" msg:"dropped_attributes_count"
	DroppedAttributesCount *uint32 `protobuf:"varint,4,opt,name=dropped_attributes_count,json=droppedAttributesCount,proto3,oneof" json:"dropped_attributes_count,omitempty"`
}

func (x *SpanEvent) Reset() {
	*x = SpanEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SpanEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SpanEvent) ProtoMessage() {}

func (x *SpanEvent) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SpanEvent.ProtoReflect.Descriptor instead.
func (*SpanEvent) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{0}
}

func (x *SpanEvent) GetTime() int64 {
	if x != nil {
		return x.Time
	}
	return 0
}

func (x *SpanEvent) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *SpanEvent) GetAttributes() map[string]*AttributeValue {
	if x != nil {
		return x.Attributes
	}
	return nil
}

func (x *SpanEvent) GetDroppedAttributesCount() uint32 {
	if x != nil && x.DroppedAttributesCount != nil {
		return *x.DroppedAttributesCount
	}
	return 0
}

// Value is either:
//   - a scalar
//   - a homogeneous array of scalars
type AttributeValue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Value:
	//
	//	*AttributeValue_Scalar
	//	*AttributeValue_StringArray
	//	*AttributeValue_BoolArray
	//	*AttributeValue_IntArray
	//	*AttributeValue_DoubleArray
	Value isAttributeValue_Value `protobuf_oneof:"value"`
}

func (x *AttributeValue) Reset() {
	*x = AttributeValue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttributeValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttributeValue) ProtoMessage() {}

func (x *AttributeValue) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttributeValue.ProtoReflect.Descriptor instead.
func (*AttributeValue) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{1}
}

func (m *AttributeValue) GetValue() isAttributeValue_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *AttributeValue) GetScalar() *AttributeValueScalar {
	if x, ok := x.GetValue().(*AttributeValue_Scalar); ok {
		return x.Scalar
	}
	return nil
}

func (x *AttributeValue) GetStringArray() *AttributeValueStringArray {
	if x, ok := x.GetValue().(*AttributeValue_StringArray); ok {
		return x.StringArray
	}
	return nil
}

func (x *AttributeValue) GetBoolArray() *AttributeValueBoolArray {
	if x, ok := x.GetValue().(*AttributeValue_BoolArray); ok {
		return x.BoolArray
	}
	return nil
}

func (x *AttributeValue) GetIntArray() *AttributeValueIntArray {
	if x, ok := x.GetValue().(*AttributeValue_IntArray); ok {
		return x.IntArray
	}
	return nil
}

func (x *AttributeValue) GetDoubleArray() *AttributeValueDoubleArray {
	if x, ok := x.GetValue().(*AttributeValue_DoubleArray); ok {
		return x.DoubleArray
	}
	return nil
}

type isAttributeValue_Value interface {
	isAttributeValue_Value()
}

type AttributeValue_Scalar struct {
	Scalar *AttributeValueScalar `protobuf:"bytes,1,opt,name=scalar,proto3,oneof"`
}

type AttributeValue_StringArray struct {
	StringArray *AttributeValueStringArray `protobuf:"bytes,3,opt,name=string_array,json=stringArray,proto3,oneof"`
}

type AttributeValue_BoolArray struct {
	BoolArray *AttributeValueBoolArray `protobuf:"bytes,4,opt,name=bool_array,json=boolArray,proto3,oneof"`
}

type AttributeValue_IntArray struct {
	IntArray *AttributeValueIntArray `protobuf:"bytes,5,opt,name=int_array,json=intArray,proto3,oneof"`
}

type AttributeValue_DoubleArray struct {
	DoubleArray *AttributeValueDoubleArray `protobuf:"bytes,6,opt,name=double_array,json=doubleArray,proto3,oneof"`
}

func (*AttributeValue_Scalar) isAttributeValue_Value() {}

func (*AttributeValue_StringArray) isAttributeValue_Value() {}

func (*AttributeValue_BoolArray) isAttributeValue_Value() {}

func (*AttributeValue_IntArray) isAttributeValue_Value() {}

func (*AttributeValue_DoubleArray) isAttributeValue_Value() {}

type AttributeValueScalar struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Value:
	//
	//	*AttributeValueScalar_String_
	//	*AttributeValueScalar_Bool
	//	*AttributeValueScalar_Int
	//	*AttributeValueScalar_Double
	Value isAttributeValueScalar_Value `protobuf_oneof:"value"`
}

func (x *AttributeValueScalar) Reset() {
	*x = AttributeValueScalar{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttributeValueScalar) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttributeValueScalar) ProtoMessage() {}

func (x *AttributeValueScalar) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttributeValueScalar.ProtoReflect.Descriptor instead.
func (*AttributeValueScalar) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{2}
}

func (m *AttributeValueScalar) GetValue() isAttributeValueScalar_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *AttributeValueScalar) GetString_() string {
	if x, ok := x.GetValue().(*AttributeValueScalar_String_); ok {
		return x.String_
	}
	return ""
}

func (x *AttributeValueScalar) GetBool() bool {
	if x, ok := x.GetValue().(*AttributeValueScalar_Bool); ok {
		return x.Bool
	}
	return false
}

func (x *AttributeValueScalar) GetInt() int64 {
	if x, ok := x.GetValue().(*AttributeValueScalar_Int); ok {
		return x.Int
	}
	return 0
}

func (x *AttributeValueScalar) GetDouble() float64 {
	if x, ok := x.GetValue().(*AttributeValueScalar_Double); ok {
		return x.Double
	}
	return 0
}

type isAttributeValueScalar_Value interface {
	isAttributeValueScalar_Value()
}

type AttributeValueScalar_String_ struct {
	String_ string `protobuf:"bytes,1,opt,name=string,proto3,oneof"`
}

type AttributeValueScalar_Bool struct {
	Bool bool `protobuf:"varint,2,opt,name=bool,proto3,oneof"`
}

type AttributeValueScalar_Int struct {
	Int int64 `protobuf:"varint,3,opt,name=int,proto3,oneof"`
}

type AttributeValueScalar_Double struct {
	Double float64 `protobuf:"fixed64,4,opt,name=double,proto3,oneof"`
}

func (*AttributeValueScalar_String_) isAttributeValueScalar_Value() {}

func (*AttributeValueScalar_Bool) isAttributeValueScalar_Value() {}

func (*AttributeValueScalar_Int) isAttributeValueScalar_Value() {}

func (*AttributeValueScalar_Double) isAttributeValueScalar_Value() {}

type AttributeValueStringArray struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value []string `protobuf:"bytes,1,rep,name=value,proto3" json:"value,omitempty"`
}

func (x *AttributeValueStringArray) Reset() {
	*x = AttributeValueStringArray{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttributeValueStringArray) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttributeValueStringArray) ProtoMessage() {}

func (x *AttributeValueStringArray) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttributeValueStringArray.ProtoReflect.Descriptor instead.
func (*AttributeValueStringArray) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{3}
}

func (x *AttributeValueStringArray) GetValue() []string {
	if x != nil {
		return x.Value
	}
	return nil
}

type AttributeValueBoolArray struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value []bool `protobuf:"varint,1,rep,packed,name=value,proto3" json:"value,omitempty"`
}

func (x *AttributeValueBoolArray) Reset() {
	*x = AttributeValueBoolArray{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttributeValueBoolArray) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttributeValueBoolArray) ProtoMessage() {}

func (x *AttributeValueBoolArray) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttributeValueBoolArray.ProtoReflect.Descriptor instead.
func (*AttributeValueBoolArray) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{4}
}

func (x *AttributeValueBoolArray) GetValue() []bool {
	if x != nil {
		return x.Value
	}
	return nil
}

type AttributeValueIntArray struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value []int64 `protobuf:"varint,1,rep,packed,name=value,proto3" json:"value,omitempty"`
}

func (x *AttributeValueIntArray) Reset() {
	*x = AttributeValueIntArray{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttributeValueIntArray) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttributeValueIntArray) ProtoMessage() {}

func (x *AttributeValueIntArray) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttributeValueIntArray.ProtoReflect.Descriptor instead.
func (*AttributeValueIntArray) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{5}
}

func (x *AttributeValueIntArray) GetValue() []int64 {
	if x != nil {
		return x.Value
	}
	return nil
}

type AttributeValueDoubleArray struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value []float64 `protobuf:"fixed64,1,rep,packed,name=value,proto3" json:"value,omitempty"`
}

func (x *AttributeValueDoubleArray) Reset() {
	*x = AttributeValueDoubleArray{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datadog_trace_span_event_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttributeValueDoubleArray) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttributeValueDoubleArray) ProtoMessage() {}

func (x *AttributeValueDoubleArray) ProtoReflect() protoreflect.Message {
	mi := &file_datadog_trace_span_event_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttributeValueDoubleArray.ProtoReflect.Descriptor instead.
func (*AttributeValueDoubleArray) Descriptor() ([]byte, []int) {
	return file_datadog_trace_span_event_proto_rawDescGZIP(), []int{6}
}

func (x *AttributeValueDoubleArray) GetValue() []float64 {
	if x != nil {
		return x.Value
	}
	return nil
}

var File_datadog_trace_span_event_proto protoreflect.FileDescriptor

var file_datadog_trace_span_event_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2f,
	0x73, 0x70, 0x61, 0x6e, 0x5f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0d, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x22,
	0xb7, 0x02, 0x0a, 0x09, 0x53, 0x70, 0x61, 0x6e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x12, 0x0a,
	0x04, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x69, 0x6d,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x48, 0x0a, 0x0a, 0x61, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75,
	0x74, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x64, 0x61, 0x74, 0x61,
	0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e, 0x53, 0x70, 0x61, 0x6e, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x2e, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x0a, 0x61, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x12,
	0x3d, 0x0a, 0x18, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x74, 0x72, 0x69,
	0x62, 0x75, 0x74, 0x65, 0x73, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0d, 0x48, 0x00, 0x52, 0x16, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x41, 0x74, 0x74, 0x72,
	0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x88, 0x01, 0x01, 0x1a, 0x5c,
	0x0a, 0x0f, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x33, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61,
	0x63, 0x65, 0x2e, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x1b, 0x0a, 0x19,
	0x5f, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75,
	0x74, 0x65, 0x73, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x85, 0x03, 0x0a, 0x0e, 0x41, 0x74,
	0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x3d, 0x0a, 0x06,
	0x73, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x64,
	0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e, 0x41, 0x74, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x53, 0x63, 0x61, 0x6c, 0x61,
	0x72, 0x48, 0x00, 0x52, 0x06, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x12, 0x4d, 0x0a, 0x0c, 0x73,
	0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x61, 0x72, 0x72, 0x61, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x28, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63,
	0x65, 0x2e, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x41, 0x72, 0x72, 0x61, 0x79, 0x48, 0x00, 0x52, 0x0b, 0x73,
	0x74, 0x72, 0x69, 0x6e, 0x67, 0x41, 0x72, 0x72, 0x61, 0x79, 0x12, 0x47, 0x0a, 0x0a, 0x62, 0x6f,
	0x6f, 0x6c, 0x5f, 0x61, 0x72, 0x72, 0x61, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26,
	0x2e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e, 0x41,
	0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x6f, 0x6f,
	0x6c, 0x41, 0x72, 0x72, 0x61, 0x79, 0x48, 0x00, 0x52, 0x09, 0x62, 0x6f, 0x6f, 0x6c, 0x41, 0x72,
	0x72, 0x61, 0x79, 0x12, 0x44, 0x0a, 0x09, 0x69, 0x6e, 0x74, 0x5f, 0x61, 0x72, 0x72, 0x61, 0x79,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67,
	0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x49, 0x6e, 0x74, 0x41, 0x72, 0x72, 0x61, 0x79, 0x48, 0x00, 0x52,
	0x08, 0x69, 0x6e, 0x74, 0x41, 0x72, 0x72, 0x61, 0x79, 0x12, 0x4d, 0x0a, 0x0c, 0x64, 0x6f, 0x75,
	0x62, 0x6c, 0x65, 0x5f, 0x61, 0x72, 0x72, 0x61, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x28, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2e, 0x74, 0x72, 0x61, 0x63, 0x65, 0x2e,
	0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x44, 0x6f,
	0x75, 0x62, 0x6c, 0x65, 0x41, 0x72, 0x72, 0x61, 0x79, 0x48, 0x00, 0x52, 0x0b, 0x64, 0x6f, 0x75,
	0x62, 0x6c, 0x65, 0x41, 0x72, 0x72, 0x61, 0x79, 0x42, 0x07, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x22, 0x7d, 0x0a, 0x14, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x12, 0x18, 0x0a, 0x06, 0x73, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x06, 0x73, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x12, 0x14, 0x0a, 0x04, 0x62, 0x6f, 0x6f, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x08, 0x48, 0x00, 0x52, 0x04, 0x62, 0x6f, 0x6f, 0x6c, 0x12, 0x12, 0x0a, 0x03, 0x69, 0x6e, 0x74,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52, 0x03, 0x69, 0x6e, 0x74, 0x12, 0x18, 0x0a,
	0x06, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x48, 0x00, 0x52,
	0x06, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x42, 0x07, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x22, 0x31, 0x0a, 0x19, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x41, 0x72, 0x72, 0x61, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x22, 0x2f, 0x0a, 0x17, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x6f, 0x6f, 0x6c, 0x41, 0x72, 0x72, 0x61, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x08, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x22, 0x2e, 0x0a, 0x16, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74,
	0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x49, 0x6e, 0x74, 0x41, 0x72, 0x72, 0x61, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x03, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x22, 0x31, 0x0a, 0x19, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74,
	0x65, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x44, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x41, 0x72, 0x72, 0x61,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x01,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x16, 0x5a, 0x14, 0x70, 0x6b, 0x67, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x62, 0x67, 0x6f, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x65, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_datadog_trace_span_event_proto_rawDescOnce sync.Once
	file_datadog_trace_span_event_proto_rawDescData = file_datadog_trace_span_event_proto_rawDesc
)

func file_datadog_trace_span_event_proto_rawDescGZIP() []byte {
	file_datadog_trace_span_event_proto_rawDescOnce.Do(func() {
		file_datadog_trace_span_event_proto_rawDescData = protoimpl.X.CompressGZIP(file_datadog_trace_span_event_proto_rawDescData)
	})
	return file_datadog_trace_span_event_proto_rawDescData
}

var file_datadog_trace_span_event_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_datadog_trace_span_event_proto_goTypes = []interface{}{
	(*SpanEvent)(nil),                 // 0: datadog.trace.SpanEvent
	(*AttributeValue)(nil),            // 1: datadog.trace.AttributeValue
	(*AttributeValueScalar)(nil),      // 2: datadog.trace.AttributeValueScalar
	(*AttributeValueStringArray)(nil), // 3: datadog.trace.AttributeValueStringArray
	(*AttributeValueBoolArray)(nil),   // 4: datadog.trace.AttributeValueBoolArray
	(*AttributeValueIntArray)(nil),    // 5: datadog.trace.AttributeValueIntArray
	(*AttributeValueDoubleArray)(nil), // 6: datadog.trace.AttributeValueDoubleArray
	nil,                               // 7: datadog.trace.SpanEvent.AttributesEntry
}
var file_datadog_trace_span_event_proto_depIdxs = []int32{
	7, // 0: datadog.trace.SpanEvent.attributes:type_name -> datadog.trace.SpanEvent.AttributesEntry
	2, // 1: datadog.trace.AttributeValue.scalar:type_name -> datadog.trace.AttributeValueScalar
	3, // 2: datadog.trace.AttributeValue.string_array:type_name -> datadog.trace.AttributeValueStringArray
	4, // 3: datadog.trace.AttributeValue.bool_array:type_name -> datadog.trace.AttributeValueBoolArray
	5, // 4: datadog.trace.AttributeValue.int_array:type_name -> datadog.trace.AttributeValueIntArray
	6, // 5: datadog.trace.AttributeValue.double_array:type_name -> datadog.trace.AttributeValueDoubleArray
	1, // 6: datadog.trace.SpanEvent.AttributesEntry.value:type_name -> datadog.trace.AttributeValue
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_datadog_trace_span_event_proto_init() }
func file_datadog_trace_span_event_proto_init() {
	if File_datadog_trace_span_event_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_datadog_trace_span_event_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SpanEvent); i {
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
		file_datadog_trace_span_event_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttributeValue); i {
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
		file_datadog_trace_span_event_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttributeValueScalar); i {
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
		file_datadog_trace_span_event_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttributeValueStringArray); i {
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
		file_datadog_trace_span_event_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttributeValueBoolArray); i {
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
		file_datadog_trace_span_event_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttributeValueIntArray); i {
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
		file_datadog_trace_span_event_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttributeValueDoubleArray); i {
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
	file_datadog_trace_span_event_proto_msgTypes[0].OneofWrappers = []interface{}{}
	file_datadog_trace_span_event_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*AttributeValue_Scalar)(nil),
		(*AttributeValue_StringArray)(nil),
		(*AttributeValue_BoolArray)(nil),
		(*AttributeValue_IntArray)(nil),
		(*AttributeValue_DoubleArray)(nil),
	}
	file_datadog_trace_span_event_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*AttributeValueScalar_String_)(nil),
		(*AttributeValueScalar_Bool)(nil),
		(*AttributeValueScalar_Int)(nil),
		(*AttributeValueScalar_Double)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_datadog_trace_span_event_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_datadog_trace_span_event_proto_goTypes,
		DependencyIndexes: file_datadog_trace_span_event_proto_depIdxs,
		MessageInfos:      file_datadog_trace_span_event_proto_msgTypes,
	}.Build()
	File_datadog_trace_span_event_proto = out.File
	file_datadog_trace_span_event_proto_rawDesc = nil
	file_datadog_trace_span_event_proto_goTypes = nil
	file_datadog_trace_span_event_proto_depIdxs = nil
}
