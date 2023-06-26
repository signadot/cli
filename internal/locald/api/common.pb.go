// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.12.4
// source: internal/locald/api/common.proto

package api

import (
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

// Service health
type ServiceHealth struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Healthy         bool                 `protobuf:"varint,1,opt,name=healthy,proto3" json:"healthy,omitempty"`
	ErrorCount      uint32               `protobuf:"varint,2,opt,name=error_count,json=errorCount,proto3" json:"error_count,omitempty"`
	LastErrorReason string               `protobuf:"bytes,3,opt,name=last_error_reason,json=lastErrorReason,proto3" json:"last_error_reason,omitempty"`
	LastErrorTime   *timestamp.Timestamp `protobuf:"bytes,4,opt,name=last_error_time,json=lastErrorTime,proto3" json:"last_error_time,omitempty"`
}

func (x *ServiceHealth) Reset() {
	*x = ServiceHealth{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServiceHealth) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServiceHealth) ProtoMessage() {}

func (x *ServiceHealth) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServiceHealth.ProtoReflect.Descriptor instead.
func (*ServiceHealth) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_common_proto_rawDescGZIP(), []int{0}
}

func (x *ServiceHealth) GetHealthy() bool {
	if x != nil {
		return x.Healthy
	}
	return false
}

func (x *ServiceHealth) GetErrorCount() uint32 {
	if x != nil {
		return x.ErrorCount
	}
	return 0
}

func (x *ServiceHealth) GetLastErrorReason() string {
	if x != nil {
		return x.LastErrorReason
	}
	return ""
}

func (x *ServiceHealth) GetLastErrorTime() *timestamp.Timestamp {
	if x != nil {
		return x.LastErrorTime
	}
	return nil
}

// LocalNet status (produced by root controller)
type LocalNetStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Health        *ServiceHealth `protobuf:"bytes,1,opt,name=health,proto3" json:"health,omitempty"`
	Cidrs         []string       `protobuf:"bytes,2,rep,name=cidrs,proto3" json:"cidrs,omitempty"`
	ExcludedCidrs []string       `protobuf:"bytes,3,rep,name=excluded_cidrs,json=excludedCidrs,proto3" json:"excluded_cidrs,omitempty"`
}

func (x *LocalNetStatus) Reset() {
	*x = LocalNetStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LocalNetStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LocalNetStatus) ProtoMessage() {}

func (x *LocalNetStatus) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LocalNetStatus.ProtoReflect.Descriptor instead.
func (*LocalNetStatus) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_common_proto_rawDescGZIP(), []int{1}
}

func (x *LocalNetStatus) GetHealth() *ServiceHealth {
	if x != nil {
		return x.Health
	}
	return nil
}

func (x *LocalNetStatus) GetCidrs() []string {
	if x != nil {
		return x.Cidrs
	}
	return nil
}

func (x *LocalNetStatus) GetExcludedCidrs() []string {
	if x != nil {
		return x.ExcludedCidrs
	}
	return nil
}

// Hosts status (produced by root controller)
type HostsStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Health         *ServiceHealth       `protobuf:"bytes,1,opt,name=health,proto3" json:"health,omitempty"`
	NumHosts       uint32               `protobuf:"varint,2,opt,name=num_hosts,json=numHosts,proto3" json:"num_hosts,omitempty"`
	NumUpdates     uint32               `protobuf:"varint,3,opt,name=num_updates,json=numUpdates,proto3" json:"num_updates,omitempty"`
	LastUpdateTime *timestamp.Timestamp `protobuf:"bytes,4,opt,name=last_update_time,json=lastUpdateTime,proto3" json:"last_update_time,omitempty"`
}

func (x *HostsStatus) Reset() {
	*x = HostsStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HostsStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HostsStatus) ProtoMessage() {}

func (x *HostsStatus) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HostsStatus.ProtoReflect.Descriptor instead.
func (*HostsStatus) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_common_proto_rawDescGZIP(), []int{2}
}

func (x *HostsStatus) GetHealth() *ServiceHealth {
	if x != nil {
		return x.Health
	}
	return nil
}

func (x *HostsStatus) GetNumHosts() uint32 {
	if x != nil {
		return x.NumHosts
	}
	return 0
}

func (x *HostsStatus) GetNumUpdates() uint32 {
	if x != nil {
		return x.NumUpdates
	}
	return 0
}

func (x *HostsStatus) GetLastUpdateTime() *timestamp.Timestamp {
	if x != nil {
		return x.LastUpdateTime
	}
	return nil
}

// PortForward status (produced by local controller)
type PortForwardStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Health       *ServiceHealth `protobuf:"bytes,1,opt,name=health,proto3" json:"health,omitempty"`
	LocalAddress string         `protobuf:"bytes,2,opt,name=local_address,json=localAddress,proto3" json:"local_address,omitempty"`
}

func (x *PortForwardStatus) Reset() {
	*x = PortForwardStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_common_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PortForwardStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PortForwardStatus) ProtoMessage() {}

func (x *PortForwardStatus) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_common_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PortForwardStatus.ProtoReflect.Descriptor instead.
func (*PortForwardStatus) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_common_proto_rawDescGZIP(), []int{3}
}

func (x *PortForwardStatus) GetHealth() *ServiceHealth {
	if x != nil {
		return x.Health
	}
	return nil
}

func (x *PortForwardStatus) GetLocalAddress() string {
	if x != nil {
		return x.LocalAddress
	}
	return ""
}

// Sandbox status (produced by local controller)
type SandboxStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name           string                         `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	RoutingKey     string                         `protobuf:"bytes,2,opt,name=routing_key,json=routingKey,proto3" json:"routing_key,omitempty"`
	LocalWorkloads []*SandboxStatus_LocalWorkload `protobuf:"bytes,3,rep,name=local_workloads,json=localWorkloads,proto3" json:"local_workloads,omitempty"`
}

func (x *SandboxStatus) Reset() {
	*x = SandboxStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_common_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxStatus) ProtoMessage() {}

func (x *SandboxStatus) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_common_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SandboxStatus.ProtoReflect.Descriptor instead.
func (*SandboxStatus) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_common_proto_rawDescGZIP(), []int{4}
}

func (x *SandboxStatus) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *SandboxStatus) GetRoutingKey() string {
	if x != nil {
		return x.RoutingKey
	}
	return ""
}

func (x *SandboxStatus) GetLocalWorkloads() []*SandboxStatus_LocalWorkload {
	if x != nil {
		return x.LocalWorkloads
	}
	return nil
}

type SandboxStatus_LocalWorkload struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name         string         `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	TunnelHealth *ServiceHealth `protobuf:"bytes,2,opt,name=tunnel_health,json=tunnelHealth,proto3" json:"tunnel_health,omitempty"`
	InClusterUrl string         `protobuf:"bytes,3,opt,name=in_cluster_url,json=inClusterUrl,proto3" json:"in_cluster_url,omitempty"`
}

func (x *SandboxStatus_LocalWorkload) Reset() {
	*x = SandboxStatus_LocalWorkload{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_locald_api_common_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SandboxStatus_LocalWorkload) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SandboxStatus_LocalWorkload) ProtoMessage() {}

func (x *SandboxStatus_LocalWorkload) ProtoReflect() protoreflect.Message {
	mi := &file_internal_locald_api_common_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SandboxStatus_LocalWorkload.ProtoReflect.Descriptor instead.
func (*SandboxStatus_LocalWorkload) Descriptor() ([]byte, []int) {
	return file_internal_locald_api_common_proto_rawDescGZIP(), []int{4, 0}
}

func (x *SandboxStatus_LocalWorkload) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *SandboxStatus_LocalWorkload) GetTunnelHealth() *ServiceHealth {
	if x != nil {
		return x.TunnelHealth
	}
	return nil
}

func (x *SandboxStatus_LocalWorkload) GetInClusterUrl() string {
	if x != nil {
		return x.InClusterUrl
	}
	return ""
}

var File_internal_locald_api_common_proto protoreflect.FileDescriptor

var file_internal_locald_api_common_proto_rawDesc = []byte{
	0x0a, 0x20, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c,
	0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x09, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x1a, 0x1f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xba,
	0x01, 0x0a, 0x0d, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68,
	0x12, 0x18, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x07, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x79, 0x12, 0x1f, 0x0a, 0x0b, 0x65, 0x72,
	0x72, 0x6f, 0x72, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x0a, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x2a, 0x0a, 0x11, 0x6c,
	0x61, 0x73, 0x74, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x6c, 0x61, 0x73, 0x74, 0x45, 0x72, 0x72, 0x6f,
	0x72, 0x52, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x42, 0x0a, 0x0f, 0x6c, 0x61, 0x73, 0x74, 0x5f,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x6c, 0x61,
	0x73, 0x74, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x54, 0x69, 0x6d, 0x65, 0x22, 0x7f, 0x0a, 0x0e, 0x4c,
	0x6f, 0x63, 0x61, 0x6c, 0x4e, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x30, 0x0a,
	0x06, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e,
	0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x52, 0x06, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x12,
	0x14, 0x0a, 0x05, 0x63, 0x69, 0x64, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05,
	0x63, 0x69, 0x64, 0x72, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65,
	0x64, 0x5f, 0x63, 0x69, 0x64, 0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0d, 0x65,
	0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x43, 0x69, 0x64, 0x72, 0x73, 0x22, 0xc3, 0x01, 0x0a,
	0x0b, 0x48, 0x6f, 0x73, 0x74, 0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x30, 0x0a, 0x06,
	0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x61,
	0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x52, 0x06, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x12, 0x1b,
	0x0a, 0x09, 0x6e, 0x75, 0x6d, 0x5f, 0x68, 0x6f, 0x73, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x08, 0x6e, 0x75, 0x6d, 0x48, 0x6f, 0x73, 0x74, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x6e,
	0x75, 0x6d, 0x5f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x0a, 0x6e, 0x75, 0x6d, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x73, 0x12, 0x44, 0x0a, 0x10,
	0x6c, 0x61, 0x73, 0x74, 0x5f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x52, 0x0e, 0x6c, 0x61, 0x73, 0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x69,
	0x6d, 0x65, 0x22, 0x6a, 0x0a, 0x11, 0x50, 0x6f, 0x72, 0x74, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72,
	0x64, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x30, 0x0a, 0x06, 0x68, 0x65, 0x61, 0x6c, 0x74,
	0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d,
	0x6d, 0x6f, 0x6e, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x48, 0x65, 0x61, 0x6c, 0x74,
	0x68, 0x52, 0x06, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x12, 0x23, 0x0a, 0x0d, 0x6c, 0x6f, 0x63,
	0x61, 0x6c, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0c, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0xa0,
	0x02, 0x0a, 0x0d, 0x53, 0x61, 0x6e, 0x64, 0x62, 0x6f, 0x78, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x5f,
	0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x6f, 0x75, 0x74, 0x69,
	0x6e, 0x67, 0x4b, 0x65, 0x79, 0x12, 0x4f, 0x0a, 0x0f, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x77,
	0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x26,
	0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x53, 0x61, 0x6e, 0x64, 0x62,
	0x6f, 0x78, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x6c, 0x57, 0x6f,
	0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x52, 0x0e, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x57, 0x6f, 0x72,
	0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x73, 0x1a, 0x88, 0x01, 0x0a, 0x0d, 0x4c, 0x6f, 0x63, 0x61, 0x6c,
	0x57, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x3d, 0x0a, 0x0d,
	0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x5f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x61, 0x70, 0x69, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x52, 0x0c, 0x74,
	0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x12, 0x24, 0x0a, 0x0e, 0x69,
	0x6e, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0c, 0x69, 0x6e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x55, 0x72,
	0x6c, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x73, 0x69, 0x67, 0x6e, 0x61, 0x64, 0x6f, 0x74, 0x2f, 0x63, 0x6c, 0x69, 0x2f, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x64, 0x2f, 0x61, 0x70, 0x69,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_locald_api_common_proto_rawDescOnce sync.Once
	file_internal_locald_api_common_proto_rawDescData = file_internal_locald_api_common_proto_rawDesc
)

func file_internal_locald_api_common_proto_rawDescGZIP() []byte {
	file_internal_locald_api_common_proto_rawDescOnce.Do(func() {
		file_internal_locald_api_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_locald_api_common_proto_rawDescData)
	})
	return file_internal_locald_api_common_proto_rawDescData
}

var file_internal_locald_api_common_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_internal_locald_api_common_proto_goTypes = []interface{}{
	(*ServiceHealth)(nil),               // 0: apicommon.ServiceHealth
	(*LocalNetStatus)(nil),              // 1: apicommon.LocalNetStatus
	(*HostsStatus)(nil),                 // 2: apicommon.HostsStatus
	(*PortForwardStatus)(nil),           // 3: apicommon.PortForwardStatus
	(*SandboxStatus)(nil),               // 4: apicommon.SandboxStatus
	(*SandboxStatus_LocalWorkload)(nil), // 5: apicommon.SandboxStatus.LocalWorkload
	(*timestamp.Timestamp)(nil),         // 6: google.protobuf.Timestamp
}
var file_internal_locald_api_common_proto_depIdxs = []int32{
	6, // 0: apicommon.ServiceHealth.last_error_time:type_name -> google.protobuf.Timestamp
	0, // 1: apicommon.LocalNetStatus.health:type_name -> apicommon.ServiceHealth
	0, // 2: apicommon.HostsStatus.health:type_name -> apicommon.ServiceHealth
	6, // 3: apicommon.HostsStatus.last_update_time:type_name -> google.protobuf.Timestamp
	0, // 4: apicommon.PortForwardStatus.health:type_name -> apicommon.ServiceHealth
	5, // 5: apicommon.SandboxStatus.local_workloads:type_name -> apicommon.SandboxStatus.LocalWorkload
	0, // 6: apicommon.SandboxStatus.LocalWorkload.tunnel_health:type_name -> apicommon.ServiceHealth
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_internal_locald_api_common_proto_init() }
func file_internal_locald_api_common_proto_init() {
	if File_internal_locald_api_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_locald_api_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ServiceHealth); i {
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
		file_internal_locald_api_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LocalNetStatus); i {
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
		file_internal_locald_api_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HostsStatus); i {
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
		file_internal_locald_api_common_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PortForwardStatus); i {
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
		file_internal_locald_api_common_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SandboxStatus); i {
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
		file_internal_locald_api_common_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SandboxStatus_LocalWorkload); i {
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
			RawDescriptor: file_internal_locald_api_common_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_locald_api_common_proto_goTypes,
		DependencyIndexes: file_internal_locald_api_common_proto_depIdxs,
		MessageInfos:      file_internal_locald_api_common_proto_msgTypes,
	}.Build()
	File_internal_locald_api_common_proto = out.File
	file_internal_locald_api_common_proto_rawDesc = nil
	file_internal_locald_api_common_proto_goTypes = nil
	file_internal_locald_api_common_proto_depIdxs = nil
}
