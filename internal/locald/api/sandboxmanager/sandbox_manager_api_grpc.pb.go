// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package sandboxmanager

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

// SandboxManagerAPIClient is the client API for SandboxManagerAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SandboxManagerAPIClient interface {
	// This method is used to create a sandbox with local references.
	// The local controller (signadot local connect) should be running,
	// otherwise it will return an error
	ApplySandbox(ctx context.Context, in *ApplySandboxRequest, opts ...grpc.CallOption) (*ApplySandboxResponse, error)
	// This method returns the status of the local controller
	Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	// This method requests the root controller to shutdown
	Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error)
}

type sandboxManagerAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewSandboxManagerAPIClient(cc grpc.ClientConnInterface) SandboxManagerAPIClient {
	return &sandboxManagerAPIClient{cc}
}

func (c *sandboxManagerAPIClient) ApplySandbox(ctx context.Context, in *ApplySandboxRequest, opts ...grpc.CallOption) (*ApplySandboxResponse, error) {
	out := new(ApplySandboxResponse)
	err := c.cc.Invoke(ctx, "/sandboxmanager.SandboxManagerAPI/ApplySandbox", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sandboxManagerAPIClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, "/sandboxmanager.SandboxManagerAPI/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sandboxManagerAPIClient) Shutdown(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error) {
	out := new(ShutdownResponse)
	err := c.cc.Invoke(ctx, "/sandboxmanager.SandboxManagerAPI/Shutdown", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SandboxManagerAPIServer is the server API for SandboxManagerAPI service.
// All implementations must embed UnimplementedSandboxManagerAPIServer
// for forward compatibility
type SandboxManagerAPIServer interface {
	// This method is used to create a sandbox with local references.
	// The local controller (signadot local connect) should be running,
	// otherwise it will return an error
	ApplySandbox(context.Context, *ApplySandboxRequest) (*ApplySandboxResponse, error)
	// This method returns the status of the local controller
	Status(context.Context, *StatusRequest) (*StatusResponse, error)
	// This method requests the root controller to shutdown
	Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error)
	mustEmbedUnimplementedSandboxManagerAPIServer()
}

// UnimplementedSandboxManagerAPIServer must be embedded to have forward compatible implementations.
type UnimplementedSandboxManagerAPIServer struct {
}

func (UnimplementedSandboxManagerAPIServer) ApplySandbox(context.Context, *ApplySandboxRequest) (*ApplySandboxResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApplySandbox not implemented")
}
func (UnimplementedSandboxManagerAPIServer) Status(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedSandboxManagerAPIServer) Shutdown(context.Context, *ShutdownRequest) (*ShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedSandboxManagerAPIServer) mustEmbedUnimplementedSandboxManagerAPIServer() {}

// UnsafeSandboxManagerAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SandboxManagerAPIServer will
// result in compilation errors.
type UnsafeSandboxManagerAPIServer interface {
	mustEmbedUnimplementedSandboxManagerAPIServer()
}

func RegisterSandboxManagerAPIServer(s grpc.ServiceRegistrar, srv SandboxManagerAPIServer) {
	s.RegisterService(&SandboxManagerAPI_ServiceDesc, srv)
}

func _SandboxManagerAPI_ApplySandbox_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ApplySandboxRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SandboxManagerAPIServer).ApplySandbox(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sandboxmanager.SandboxManagerAPI/ApplySandbox",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SandboxManagerAPIServer).ApplySandbox(ctx, req.(*ApplySandboxRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SandboxManagerAPI_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SandboxManagerAPIServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sandboxmanager.SandboxManagerAPI/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SandboxManagerAPIServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SandboxManagerAPI_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SandboxManagerAPIServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sandboxmanager.SandboxManagerAPI/Shutdown",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SandboxManagerAPIServer).Shutdown(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SandboxManagerAPI_ServiceDesc is the grpc.ServiceDesc for SandboxManagerAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SandboxManagerAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sandboxmanager.SandboxManagerAPI",
	HandlerType: (*SandboxManagerAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ApplySandbox",
			Handler:    _SandboxManagerAPI_ApplySandbox_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _SandboxManagerAPI_Status_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _SandboxManagerAPI_Shutdown_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/locald/api/sandboxmanager/sandbox_manager_api.proto",
}
