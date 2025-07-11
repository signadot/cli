// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v5.28.2
// source: internal/locald/api/rootmanager/root_manager_api.proto

package rootmanager

import (
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
		mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusRequest) ProtoMessage() {}

func (x *StatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[0]
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
	return file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescGZIP(), []int{0}
}

type StatusResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Localnet *api.LocalNetStatus `protobuf:"bytes,1,opt,name=localnet,proto3" json:"localnet,omitempty"`
	Hosts    *api.HostsStatus    `protobuf:"bytes,2,opt,name=hosts,proto3" json:"hosts,omitempty"`
}

func (x *StatusResponse) Reset() {
	*x = StatusResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResponse) ProtoMessage() {}

func (x *StatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[1]
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
	return file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescGZIP(), []int{1}
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

type ShutdownRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ShutdownRequest) Reset() {
	*x = ShutdownRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShutdownRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownRequest) ProtoMessage() {}

func (x *ShutdownRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[2]
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
	return file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescGZIP(), []int{2}
}

type ShutdownResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ShutdownResponse) Reset() {
	*x = ShutdownResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ShutdownResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownResponse) ProtoMessage() {}

func (x *ShutdownResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[3]
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
	return file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescGZIP(), []int{3}
}

var File_internal_locald_api_rootmanager_root_manager_api_proto protoreflect.FileDescriptor

var file_internal_locald_api_rootmanager_root_manager_api_proto_rawDesc = []byte{
	0x0a, 0x36, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c,
	0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x72, 0x6f, 0x6f, 0x74, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65,
	0x72, 0x2f, 0x72, 0x6f, 0x6f, 0x74, 0x5f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x5f, 0x61,
	0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x72, 0x6f, 0x6f, 0x74, 0x6d, 0x61,
	0x6e, 0x61, 0x67, 0x65, 0x72, 0x1a, 0x20, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f,
	0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x0f, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x75, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x35, 0x0a, 0x08, 0x6c, 0x6f,
	0x63, 0x61, 0x6c, 0x6e, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x61,
	0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x6c, 0x4e, 0x65,
	0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x08, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x6e, 0x65,
	0x74, 0x12, 0x2c, 0x0a, 0x05, 0x68, 0x6f, 0x73, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x16, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x48, 0x6f, 0x73,
	0x74, 0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x05, 0x68, 0x6f, 0x73, 0x74, 0x73, 0x22,
	0x11, 0x0a, 0x0f, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x22, 0x12, 0x0a, 0x10, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0xa0, 0x01, 0x0a, 0x0e, 0x52, 0x6f, 0x6f, 0x74, 0x4d,
	0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x41, 0x50, 0x49, 0x12, 0x43, 0x0a, 0x06, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x1a, 0x2e, 0x72, 0x6f, 0x6f, 0x74, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65,
	0x72, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x1b, 0x2e, 0x72, 0x6f, 0x6f, 0x74, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x49,
	0x0a, 0x08, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x1c, 0x2e, 0x72, 0x6f, 0x6f,
	0x74, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77,
	0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e, 0x72, 0x6f, 0x6f, 0x74, 0x6d,
	0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x39, 0x5a, 0x37, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x64, 0x6f, 0x74,
	0x2f, 0x63, 0x6c, 0x69, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6c, 0x6f,
	0x63, 0x61, 0x6c, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x72, 0x6f, 0x6f, 0x74, 0x6d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescOnce sync.Once
	file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescData = file_internal_locald_api_rootmanager_root_manager_api_proto_rawDesc
)

func file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescGZIP() []byte {
	file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescOnce.Do(func() {
		file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescData)
	})
	return file_internal_locald_api_rootmanager_root_manager_api_proto_rawDescData
}

var file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_internal_locald_api_rootmanager_root_manager_api_proto_goTypes = []interface{}{
	(*StatusRequest)(nil),      // 0: rootmanager.StatusRequest
	(*StatusResponse)(nil),     // 1: rootmanager.StatusResponse
	(*ShutdownRequest)(nil),    // 2: rootmanager.ShutdownRequest
	(*ShutdownResponse)(nil),   // 3: rootmanager.ShutdownResponse
	(*api.LocalNetStatus)(nil), // 4: apicommon.LocalNetStatus
	(*api.HostsStatus)(nil),    // 5: apicommon.HostsStatus
}
var file_internal_locald_api_rootmanager_root_manager_api_proto_depIdxs = []int32{
	4, // 0: rootmanager.StatusResponse.localnet:type_name -> apicommon.LocalNetStatus
	5, // 1: rootmanager.StatusResponse.hosts:type_name -> apicommon.HostsStatus
	0, // 2: rootmanager.RootManagerAPI.Status:input_type -> rootmanager.StatusRequest
	2, // 3: rootmanager.RootManagerAPI.Shutdown:input_type -> rootmanager.ShutdownRequest
	1, // 4: rootmanager.RootManagerAPI.Status:output_type -> rootmanager.StatusResponse
	3, // 5: rootmanager.RootManagerAPI.Shutdown:output_type -> rootmanager.ShutdownResponse
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_internal_locald_api_rootmanager_root_manager_api_proto_init() }
func file_internal_locald_api_rootmanager_root_manager_api_proto_init() {
	if File_internal_locald_api_rootmanager_root_manager_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
		file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
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
			RawDescriptor: file_internal_locald_api_rootmanager_root_manager_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_locald_api_rootmanager_root_manager_api_proto_goTypes,
		DependencyIndexes: file_internal_locald_api_rootmanager_root_manager_api_proto_depIdxs,
		MessageInfos:      file_internal_locald_api_rootmanager_root_manager_api_proto_msgTypes,
	}.Build()
	File_internal_locald_api_rootmanager_root_manager_api_proto = out.File
	file_internal_locald_api_rootmanager_root_manager_api_proto_rawDesc = nil
	file_internal_locald_api_rootmanager_root_manager_api_proto_goTypes = nil
	file_internal_locald_api_rootmanager_root_manager_api_proto_depIdxs = nil
}
