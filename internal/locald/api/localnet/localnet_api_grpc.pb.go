// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.12.4
// source: internal/locald/api/localnet/localnet_api.proto

package localnet

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	LocalnetAPI_Status_FullMethodName   = "/localnet.LocalnetAPI/Status"
	LocalnetAPI_Shutdown_FullMethodName = "/localnet.LocalnetAPI/Shutdown"
)

// LocalnetAPIClient is the client API for LocalnetAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LocalnetAPIClient interface {
	// This method returns the status of the root controller
	Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	// This method requests the root controller to shutdown
	Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error)
}

type localnetAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewLocalnetAPIClient(cc grpc.ClientConnInterface) LocalnetAPIClient {
	return &localnetAPIClient{cc}
}

func (c *localnetAPIClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, LocalnetAPI_Status_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *localnetAPIClient) Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error) {
	out := new(ShutdownResponse)
	err := c.cc.Invoke(ctx, LocalnetAPI_Shutdown_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LocalnetAPIServer is the server API for LocalnetAPI service.
// All implementations must embed UnimplementedLocalnetAPIServer
// for forward compatibility
type LocalnetAPIServer interface {
	// This method returns the status of the root controller
	Status(context.Context, *StatusRequest) (*StatusResponse, error)
	// This method requests the root controller to shutdown
	Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error)
	mustEmbedUnimplementedLocalnetAPIServer()
}

// UnimplementedLocalnetAPIServer must be embedded to have forward compatible implementations.
type UnimplementedLocalnetAPIServer struct {
}

func (UnimplementedLocalnetAPIServer) Status(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedLocalnetAPIServer) Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedLocalnetAPIServer) mustEmbedUnimplementedLocalnetAPIServer() {}

// UnsafeLocalnetAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LocalnetAPIServer will
// result in compilation errors.
type UnsafeLocalnetAPIServer interface {
	mustEmbedUnimplementedLocalnetAPIServer()
}

func RegisterLocalnetAPIServer(s grpc.ServiceRegistrar, srv LocalnetAPIServer) {
	s.RegisterService(&LocalnetAPI_ServiceDesc, srv)
}

func _LocalnetAPI_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocalnetAPIServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LocalnetAPI_Status_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocalnetAPIServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LocalnetAPI_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocalnetAPIServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LocalnetAPI_Shutdown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocalnetAPIServer).Shutdown(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// LocalnetAPI_ServiceDesc is the grpc.ServiceDesc for LocalnetAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LocalnetAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "localnet.LocalnetAPI",
	HandlerType: (*LocalnetAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Status",
			Handler:    _LocalnetAPI_Status_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _LocalnetAPI_Shutdown_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/locald/api/localnet/localnet_api.proto",
}
