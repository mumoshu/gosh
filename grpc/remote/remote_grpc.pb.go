// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package remote

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

// RemoteClient is the client API for Remote service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RemoteClient interface {
	ShellSession(ctx context.Context, opts ...grpc.CallOption) (Remote_ShellSessionClient, error)
}

type remoteClient struct {
	cc grpc.ClientConnInterface
}

func NewRemoteClient(cc grpc.ClientConnInterface) RemoteClient {
	return &remoteClient{cc}
}

func (c *remoteClient) ShellSession(ctx context.Context, opts ...grpc.CallOption) (Remote_ShellSessionClient, error) {
	stream, err := c.cc.NewStream(ctx, &Remote_ServiceDesc.Streams[0], "/remote.Remote/ShellSession", opts...)
	if err != nil {
		return nil, err
	}
	x := &remoteShellSessionClient{stream}
	return x, nil
}

type Remote_ShellSessionClient interface {
	Send(*Message) error
	Recv() (*Message, error)
	grpc.ClientStream
}

type remoteShellSessionClient struct {
	grpc.ClientStream
}

func (x *remoteShellSessionClient) Send(m *Message) error {
	return x.ClientStream.SendMsg(m)
}

func (x *remoteShellSessionClient) Recv() (*Message, error) {
	m := new(Message)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// RemoteServer is the server API for Remote service.
// All implementations must embed UnimplementedRemoteServer
// for forward compatibility
type RemoteServer interface {
	ShellSession(Remote_ShellSessionServer) error
	mustEmbedUnimplementedRemoteServer()
}

// UnimplementedRemoteServer must be embedded to have forward compatible implementations.
type UnimplementedRemoteServer struct {
}

func (UnimplementedRemoteServer) ShellSession(Remote_ShellSessionServer) error {
	return status.Errorf(codes.Unimplemented, "method ShellSession not implemented")
}
func (UnimplementedRemoteServer) mustEmbedUnimplementedRemoteServer() {}

// UnsafeRemoteServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RemoteServer will
// result in compilation errors.
type UnsafeRemoteServer interface {
	mustEmbedUnimplementedRemoteServer()
}

func RegisterRemoteServer(s grpc.ServiceRegistrar, srv RemoteServer) {
	s.RegisterService(&Remote_ServiceDesc, srv)
}

func _Remote_ShellSession_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(RemoteServer).ShellSession(&remoteShellSessionServer{stream})
}

type Remote_ShellSessionServer interface {
	Send(*Message) error
	Recv() (*Message, error)
	grpc.ServerStream
}

type remoteShellSessionServer struct {
	grpc.ServerStream
}

func (x *remoteShellSessionServer) Send(m *Message) error {
	return x.ServerStream.SendMsg(m)
}

func (x *remoteShellSessionServer) Recv() (*Message, error) {
	m := new(Message)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Remote_ServiceDesc is the grpc.ServiceDesc for Remote service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Remote_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "remote.Remote",
	HandlerType: (*RemoteServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ShellSession",
			Handler:       _Remote_ShellSession_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "remote/remote.proto",
}