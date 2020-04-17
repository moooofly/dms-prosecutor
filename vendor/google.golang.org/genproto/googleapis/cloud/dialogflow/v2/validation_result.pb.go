// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/cloud/dialogflow/v2/validation_result.proto

package dialogflow

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// Represents a level of severity.
type ValidationError_Severity int32

const (
	// Not specified. This value should never be used.
	ValidationError_SEVERITY_UNSPECIFIED ValidationError_Severity = 0
	// The agent doesn't follow Dialogflow best practicies.
	ValidationError_INFO ValidationError_Severity = 1
	// The agent may not behave as expected.
	ValidationError_WARNING ValidationError_Severity = 2
	// The agent may experience partial failures.
	ValidationError_ERROR ValidationError_Severity = 3
	// The agent may completely fail.
	ValidationError_CRITICAL ValidationError_Severity = 4
)

var ValidationError_Severity_name = map[int32]string{
	0: "SEVERITY_UNSPECIFIED",
	1: "INFO",
	2: "WARNING",
	3: "ERROR",
	4: "CRITICAL",
}

var ValidationError_Severity_value = map[string]int32{
	"SEVERITY_UNSPECIFIED": 0,
	"INFO":                 1,
	"WARNING":              2,
	"ERROR":                3,
	"CRITICAL":             4,
}

func (x ValidationError_Severity) String() string {
	return proto.EnumName(ValidationError_Severity_name, int32(x))
}

func (ValidationError_Severity) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_7da0ebf221ed64af, []int{0, 0}
}

// Represents a single validation error.
type ValidationError struct {
	// The severity of the error.
	Severity ValidationError_Severity `protobuf:"varint,1,opt,name=severity,proto3,enum=google.cloud.dialogflow.v2.ValidationError_Severity" json:"severity,omitempty"`
	// The names of the entries that the error is associated with.
	// Format:
	//
	// - "projects/<Project ID>/agent", if the error is associated with the entire
	// agent.
	// - "projects/<Project ID>/agent/intents/<Intent ID>", if the error is
	// associated with certain intents.
	// - "projects/<Project
	// ID>/agent/intents/<Intent Id>/trainingPhrases/<Training Phrase ID>", if the
	// error is associated with certain intent training phrases.
	// - "projects/<Project ID>/agent/intents/<Intent Id>/parameters/<Parameter
	// ID>", if the error is associated with certain intent parameters.
	// - "projects/<Project ID>/agent/entities/<Entity ID>", if the error is
	// associated with certain entities.
	Entries []string `protobuf:"bytes,3,rep,name=entries,proto3" json:"entries,omitempty"`
	// The detailed error messsage.
	ErrorMessage         string   `protobuf:"bytes,4,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ValidationError) Reset()         { *m = ValidationError{} }
func (m *ValidationError) String() string { return proto.CompactTextString(m) }
func (*ValidationError) ProtoMessage()    {}
func (*ValidationError) Descriptor() ([]byte, []int) {
	return fileDescriptor_7da0ebf221ed64af, []int{0}
}

func (m *ValidationError) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ValidationError.Unmarshal(m, b)
}
func (m *ValidationError) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ValidationError.Marshal(b, m, deterministic)
}
func (m *ValidationError) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ValidationError.Merge(m, src)
}
func (m *ValidationError) XXX_Size() int {
	return xxx_messageInfo_ValidationError.Size(m)
}
func (m *ValidationError) XXX_DiscardUnknown() {
	xxx_messageInfo_ValidationError.DiscardUnknown(m)
}

var xxx_messageInfo_ValidationError proto.InternalMessageInfo

func (m *ValidationError) GetSeverity() ValidationError_Severity {
	if m != nil {
		return m.Severity
	}
	return ValidationError_SEVERITY_UNSPECIFIED
}

func (m *ValidationError) GetEntries() []string {
	if m != nil {
		return m.Entries
	}
	return nil
}

func (m *ValidationError) GetErrorMessage() string {
	if m != nil {
		return m.ErrorMessage
	}
	return ""
}

// Represents the output of agent validation.
type ValidationResult struct {
	// Contains all validation errors.
	ValidationErrors     []*ValidationError `protobuf:"bytes,1,rep,name=validation_errors,json=validationErrors,proto3" json:"validation_errors,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *ValidationResult) Reset()         { *m = ValidationResult{} }
func (m *ValidationResult) String() string { return proto.CompactTextString(m) }
func (*ValidationResult) ProtoMessage()    {}
func (*ValidationResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_7da0ebf221ed64af, []int{1}
}

func (m *ValidationResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ValidationResult.Unmarshal(m, b)
}
func (m *ValidationResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ValidationResult.Marshal(b, m, deterministic)
}
func (m *ValidationResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ValidationResult.Merge(m, src)
}
func (m *ValidationResult) XXX_Size() int {
	return xxx_messageInfo_ValidationResult.Size(m)
}
func (m *ValidationResult) XXX_DiscardUnknown() {
	xxx_messageInfo_ValidationResult.DiscardUnknown(m)
}

var xxx_messageInfo_ValidationResult proto.InternalMessageInfo

func (m *ValidationResult) GetValidationErrors() []*ValidationError {
	if m != nil {
		return m.ValidationErrors
	}
	return nil
}

func init() {
	proto.RegisterEnum("google.cloud.dialogflow.v2.ValidationError_Severity", ValidationError_Severity_name, ValidationError_Severity_value)
	proto.RegisterType((*ValidationError)(nil), "google.cloud.dialogflow.v2.ValidationError")
	proto.RegisterType((*ValidationResult)(nil), "google.cloud.dialogflow.v2.ValidationResult")
}

func init() {
	proto.RegisterFile("google/cloud/dialogflow/v2/validation_result.proto", fileDescriptor_7da0ebf221ed64af)
}

var fileDescriptor_7da0ebf221ed64af = []byte{
	// 382 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x5f, 0xab, 0xd3, 0x30,
	0x18, 0xc6, 0x4d, 0x3b, 0x3d, 0x5d, 0xce, 0x51, 0x6b, 0x50, 0x28, 0x43, 0xa4, 0xcc, 0x9b, 0x82,
	0x90, 0x42, 0xf5, 0xce, 0xab, 0x73, 0xda, 0x6e, 0x14, 0xb4, 0x2b, 0xe9, 0x9c, 0x7f, 0x6e, 0x46,
	0xdc, 0x62, 0x28, 0x64, 0xcd, 0x48, 0xba, 0x8a, 0x5f, 0x47, 0xbc, 0xf2, 0x13, 0xee, 0x52, 0x96,
	0xce, 0x55, 0x87, 0x0a, 0x5e, 0xbe, 0x6f, 0xf9, 0xfd, 0xfa, 0xf0, 0xbc, 0x81, 0x11, 0x97, 0x92,
	0x0b, 0x16, 0xae, 0x84, 0xdc, 0xad, 0xc3, 0x75, 0x45, 0x85, 0xe4, 0x9f, 0x84, 0xfc, 0x1c, 0xb6,
	0x51, 0xd8, 0x52, 0x51, 0xad, 0x69, 0x53, 0xc9, 0x7a, 0xa9, 0x98, 0xde, 0x89, 0x06, 0x6f, 0x95,
	0x6c, 0x24, 0x1a, 0x75, 0x0c, 0x36, 0x0c, 0xee, 0x19, 0xdc, 0x46, 0xa3, 0xc7, 0x47, 0x1f, 0xdd,
	0x56, 0x21, 0xad, 0x6b, 0xd9, 0x18, 0x5e, 0x77, 0xe4, 0x78, 0x0f, 0xe0, 0xfd, 0xc5, 0xc9, 0x9a,
	0x2a, 0x25, 0x15, 0x2a, 0xa0, 0xa3, 0x59, 0xcb, 0x54, 0xd5, 0x7c, 0xf1, 0x80, 0x0f, 0x82, 0x7b,
	0xd1, 0x0b, 0xfc, 0xf7, 0x1f, 0xe0, 0x33, 0x1c, 0x97, 0x47, 0x96, 0x9c, 0x2c, 0xc8, 0x83, 0x17,
	0xac, 0x6e, 0x54, 0xc5, 0xb4, 0x67, 0xfb, 0x76, 0x30, 0x24, 0x3f, 0x47, 0xf4, 0x14, 0xde, 0x65,
	0x07, 0x6a, 0xb9, 0x61, 0x5a, 0x53, 0xce, 0xbc, 0x81, 0x0f, 0x82, 0x21, 0xb9, 0x32, 0xcb, 0xd7,
	0xdd, 0x6e, 0x3c, 0x87, 0x4e, 0xd9, 0xab, 0x1e, 0x96, 0xe9, 0x22, 0x25, 0xd9, 0xfc, 0xfd, 0xf2,
	0x4d, 0x5e, 0x16, 0x69, 0x9c, 0x4d, 0xb2, 0x34, 0x71, 0x6f, 0x21, 0x07, 0x0e, 0xb2, 0x7c, 0x32,
	0x73, 0x01, 0xba, 0x84, 0x17, 0x6f, 0xaf, 0x49, 0x9e, 0xe5, 0x53, 0xd7, 0x42, 0x43, 0x78, 0x3b,
	0x25, 0x64, 0x46, 0x5c, 0x1b, 0x5d, 0x41, 0x27, 0x26, 0xd9, 0x3c, 0x8b, 0xaf, 0x5f, 0xb9, 0x83,
	0xb1, 0x80, 0x6e, 0x1f, 0x9d, 0x98, 0x3a, 0xd1, 0x3b, 0xf8, 0xe0, 0x97, 0x8e, 0x4d, 0x08, 0xed,
	0x01, 0xdf, 0x0e, 0x2e, 0xa3, 0x67, 0xff, 0xd1, 0x01, 0x71, 0xdb, 0xdf, 0x17, 0xfa, 0xe6, 0x1b,
	0x80, 0x4f, 0x56, 0x72, 0xf3, 0x0f, 0xc9, 0xcd, 0xa3, 0xf3, 0x38, 0xc5, 0xe1, 0x44, 0x05, 0xf8,
	0x90, 0x1c, 0x21, 0x2e, 0x05, 0xad, 0x39, 0x96, 0x8a, 0x87, 0x9c, 0xd5, 0xe6, 0x80, 0x61, 0xf7,
	0x89, 0x6e, 0x2b, 0xfd, 0xa7, 0x17, 0xf3, 0xb2, 0x9f, 0xf6, 0x00, 0x7c, 0xb5, 0xac, 0x64, 0xf2,
	0xdd, 0x1a, 0x4d, 0x3b, 0x5d, 0x6c, 0x32, 0x24, 0x7d, 0x86, 0x45, 0xf4, 0xf1, 0x8e, 0xb1, 0x3e,
	0xff, 0x11, 0x00, 0x00, 0xff, 0xff, 0x63, 0x7c, 0x01, 0x70, 0x86, 0x02, 0x00, 0x00,
}
