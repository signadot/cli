// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.12.4
// source: internal/locald/api/sandboxmanager/sandbox_manager_api.proto

package sandboxmanager

import (
	_struct "github.com/golang/protobuf/ptypes/struct"
	api "github.com/signadot/cli/internal/locald/api"
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

type StatusRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StatusRequest) Reset() {
	*x = StatusRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusRequest) ProtoMessage() {}

func (x *StatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusRequest.ProtoReflect.Descriptor instead.
func (*StatusRequest) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescGZIP(), []int{0}
}

type StatusResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// connect invocation config
	// (instance of internal/config/locald.ConnectInvocationConfig)
	CiConfig    *_struct.Struct        `protobuf:"bytes,1,opt,name=ci_config,json=ciConfig,proto3" json:"ci_config,omitempty"`
	Localnet    *api.LocalNetStatus    `protobuf:"bytes,2,opt,name=localnet,proto3" json:"localnet,omitempty"`
	Hosts       *api.HostsStatus       `protobuf:"bytes,3,opt,name=hosts,proto3" json:"hosts,omitempty"`
	Portforward *api.PortForwardStatus `protobuf:"bytes,4,opt,name=portforward,proto3" json:"portforward,omitempty"`
	Sandboxes   []*api.SandboxStatus   `protobuf:"bytes,5,rep,name=sandboxes,proto3" json:"sandboxes,omitempty"`
}

func (x *StatusResponse) Reset() {
	*x = StatusResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResponse) ProtoMessage() {}

func (x *StatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusResponse.ProtoReflect.Descriptor instead.
func (*StatusResponse) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescGZIP(), []int{1}
}

func (x *StatusResponse) GetCiConfig() *_struct.Struct {
	if x != nil {
		return x.CiConfig
	}
	return nil
}

func (x *StatusResponse) GetLocalnet() *api.LocalNetStatus {
	if x != nil {
		return x.Localnet
	}
	return nil
}

func (x *StatusResponse) GetHosts() *api.HostsStatus {
	if x != nil {
		return x.Hosts
	}
	return nil
}

func (x *StatusResponse) GetPortforward() *api.PortForwardStatus {
	if x != nil {
		return x.Portforward
	}
	return nil
}

func (x *StatusResponse) GetSandboxes() []*api.SandboxStatus {
	if x != nil {
		return x.Sandboxes
	}
	return nil
}

type ShutdownRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ShutdownRequest) Reset() {
	*x = ShutdownRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShutdownRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownRequest) ProtoMessage() {}

func (x *ShutdownRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShutdownRequest.ProtoReflect.Descriptor instead.
func (*ShutdownRequest) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescGZIP(), []int{2}
}

type ShutdownResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ShutdownResponse) Reset() {
	*x = ShutdownResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShutdownResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownResponse) ProtoMessage() {}

func (x *ShutdownResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ShutdownResponse.ProtoReflect.Descriptor instead.
func (*ShutdownResponse) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescGZIP(), []int{3}
}

var File_internal_locald_api_sandboxmanager_sandbox_manager_api_proto protoreflect.FileDescriptor

var file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c,
	0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x72, 0x2f, 0x73, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x5f, 0x6d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x72, 0x5f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e,
	0x73, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x1a, 0x1c,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x64, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x0f,
	0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22,
	0xa3, 0x02, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x34, 0x0a, 0x09, 0x63, 0x69, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x08,
	0x63, 0x69, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x35, 0x0a, 0x08, 0x6c, 0x6f, 0x63, 0x61,
	0x6c, 0x6e, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x61, 0x70, 0x69,
	0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x6c, 0x4e, 0x65, 0x74, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x08, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x6e, 0x65, 0x74, 0x12,
	0x2c, 0x0a, 0x05, 0x68, 0x6f, 0x73, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16,
	0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x48, 0x6f, 0x73, 0x74, 0x73,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x05, 0x68, 0x6f, 0x73, 0x74, 0x73, 0x12, 0x3e, 0x0a,
	0x0b, 0x70, 0x6f, 0x72, 0x74, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x50,
	0x6f, 0x72, 0x74, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x0b, 0x70, 0x6f, 0x72, 0x74, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x12, 0x36, 0x0a,
	0x09, 0x73, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x18, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x53, 0x61, 0x6e,
	0x64, 0x62, 0x6f, 0x78, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x09, 0x73, 0x61, 0x6e, 0x64,
	0x62, 0x6f, 0x78, 0x65, 0x73, 0x22, 0x11, 0x0a, 0x0f, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77,
	0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x12, 0x0a, 0x10, 0x53, 0x68, 0x75, 0x74,
	0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0xaf, 0x01, 0x0a,
	0x11, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x41,
	0x50, 0x49, 0x12, 0x49, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1d, 0x2e, 0x73,
	0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x2e, 0x73, 0x61,
	0x6e, 0x64, 0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x4f, 0x0a,
	0x08, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x1f, 0x2e, 0x73, 0x61, 0x6e, 0x64,
	0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64,
	0x6f, 0x77, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x20, 0x2e, 0x73, 0x61, 0x6e,
	0x64, 0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x68, 0x75, 0x74,
	0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x3c,
	0x5a, 0x3a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x69, 0x67,
	0x6e, 0x61, 0x64, 0x6f, 0x74, 0x2f, 0x63, 0x6c, 0x69, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x61,
	0x6e, 0x64, 0x62, 0x6f, 0x78, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescOnce sync.Once
	file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescData = file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDesc
)

func file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescGZIP() []byte {
	file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescOnce.Do(func() {
		file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescData)
	})
	return file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDescData
}

var file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_goTypes = []interface{}{
	(*StatusRequest)(nil),         // 0: sandboxmanager.StatusRequest
	(*StatusResponse)(nil),        // 1: sandboxmanager.StatusResponse
	(*ShutdownRequest)(nil),       // 2: sandboxmanager.ShutdownRequest
	(*ShutdownResponse)(nil),      // 3: sandboxmanager.ShutdownResponse
	(*_struct.Struct)(nil),        // 4: google.protobuf.Struct
	(*api.LocalNetStatus)(nil),    // 5: apicommon.LocalNetStatus
	(*api.HostsStatus)(nil),       // 6: apicommon.HostsStatus
	(*api.PortForwardStatus)(nil), // 7: apicommon.PortForwardStatus
	(*api.SandboxStatus)(nil),     // 8: apicommon.SandboxStatus
}
var file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_depIdxs = []int32{
	4, // 0: sandboxmanager.StatusResponse.ci_config:type_name -> google.protobuf.Struct
	5, // 1: sandboxmanager.StatusResponse.localnet:type_name -> apicommon.LocalNetStatus
	6, // 2: sandboxmanager.StatusResponse.hosts:type_name -> apicommon.HostsStatus
	7, // 3: sandboxmanager.StatusResponse.portforward:type_name -> apicommon.PortForwardStatus
	8, // 4: sandboxmanager.StatusResponse.sandboxes:type_name -> apicommon.SandboxStatus
	0, // 5: sandboxmanager.SandboxManagerAPI.Status:input_type -> sandboxmanager.StatusRequest
	2, // 6: sandboxmanager.SandboxManagerAPI.Shutdown:input_type -> sandboxmanager.ShutdownRequest
	1, // 7: sandboxmanager.SandboxManagerAPI.Status:output_type -> sandboxmanager.StatusResponse
	3, // 8: sandboxmanager.SandboxManagerAPI.Shutdown:output_type -> sandboxmanager.ShutdownResponse
	7, // [7:9] is the sub-list for method output_type
	5, // [5:7] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_init() }
func file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_init() {
	if File_internal_locald_api_sandboxmanager_sandbox_manager_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusRequest); i {
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
		file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusResponse); i {
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
		file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShutdownRequest); i {
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
		file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ShutdownResponse); i {
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
			RawDescriptor: file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_goTypes,
		DependencyIndexes: file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_depIdxs,
		MessageInfos:      file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_msgTypes,
	}.Build()
	File_internal_locald_api_sandboxmanager_sandbox_manager_api_proto = out.File
	file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_rawDesc = nil
	file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_goTypes = nil
	file_internal_locald_api_sandboxmanager_sandbox_manager_api_proto_depIdxs = nil
}
