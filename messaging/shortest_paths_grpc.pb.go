// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package messaging

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// PathMessengerClient is the client API for PathMessenger service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PathMessengerClient interface {
	//Starts the PathMessenger nodes' tasks.
	StartTask(ctx context.Context, in *TaskRequest, opts ...grpc.CallOption) (*TaskConfirmation, error)
	//Either relays the message another hop toward its destination or processes the payload value if the node is the destination.
	AcceptMessage(ctx context.Context, in *PathMessage, opts ...grpc.CallOption) (*PathResponse, error)
	//Transmits metadata about the messages the node has sent and received over the course of the task.
	GetMessagingData(ctx context.Context, in *MessagingDataRequest, opts ...grpc.CallOption) (*MessagingMetadata, error)
	//Sends a stream of Edges that represent the overlay the node is part of.
	PushPaths(ctx context.Context, opts ...grpc.CallOption) (PathMessenger_PushPathsClient, error)
}

type pathMessengerClient struct {
	cc grpc.ClientConnInterface
}

func NewPathMessengerClient(cc grpc.ClientConnInterface) PathMessengerClient {
	return &pathMessengerClient{cc}
}

func (c *pathMessengerClient) StartTask(ctx context.Context, in *TaskRequest, opts ...grpc.CallOption) (*TaskConfirmation, error) {
	out := new(TaskConfirmation)
	err := c.cc.Invoke(ctx, "/messaging.PathMessenger/StartTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pathMessengerClient) AcceptMessage(ctx context.Context, in *PathMessage, opts ...grpc.CallOption) (*PathResponse, error) {
	out := new(PathResponse)
	err := c.cc.Invoke(ctx, "/messaging.PathMessenger/AcceptMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pathMessengerClient) GetMessagingData(ctx context.Context, in *MessagingDataRequest, opts ...grpc.CallOption) (*MessagingMetadata, error) {
	out := new(MessagingMetadata)
	err := c.cc.Invoke(ctx, "/messaging.PathMessenger/GetMessagingData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pathMessengerClient) PushPaths(ctx context.Context, opts ...grpc.CallOption) (PathMessenger_PushPathsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_PathMessenger_serviceDesc.Streams[0], "/messaging.PathMessenger/PushPaths", opts...)
	if err != nil {
		return nil, err
	}
	x := &pathMessengerPushPathsClient{stream}
	return x, nil
}

type PathMessenger_PushPathsClient interface {
	Send(*Edge) error
	CloseAndRecv() (*ConnectionResponse, error)
	grpc.ClientStream
}

type pathMessengerPushPathsClient struct {
	grpc.ClientStream
}

func (x *pathMessengerPushPathsClient) Send(m *Edge) error {
	return x.ClientStream.SendMsg(m)
}

func (x *pathMessengerPushPathsClient) CloseAndRecv() (*ConnectionResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(ConnectionResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// PathMessengerServer is the server API for PathMessenger service.
// All implementations must embed UnimplementedPathMessengerServer
// for forward compatibility
type PathMessengerServer interface {
	//Starts the PathMessenger nodes' tasks.
	StartTask(context.Context, *TaskRequest) (*TaskConfirmation, error)
	//Either relays the message another hop toward its destination or processes the payload value if the node is the destination.
	AcceptMessage(context.Context, *PathMessage) (*PathResponse, error)
	//Transmits metadata about the messages the node has sent and received over the course of the task.
	GetMessagingData(context.Context, *MessagingDataRequest) (*MessagingMetadata, error)
	//Sends a stream of Edges that represent the overlay the node is part of.
	PushPaths(PathMessenger_PushPathsServer) error
	mustEmbedUnimplementedPathMessengerServer()
}

// UnimplementedPathMessengerServer must be embedded to have forward compatible implementations.
type UnimplementedPathMessengerServer struct {
}

func (UnimplementedPathMessengerServer) StartTask(context.Context, *TaskRequest) (*TaskConfirmation, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartTask not implemented")
}
func (UnimplementedPathMessengerServer) AcceptMessage(context.Context, *PathMessage) (*PathResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AcceptMessage not implemented")
}
func (UnimplementedPathMessengerServer) GetMessagingData(context.Context, *MessagingDataRequest) (*MessagingMetadata, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMessagingData not implemented")
}
func (UnimplementedPathMessengerServer) PushPaths(PathMessenger_PushPathsServer) error {
	return status.Errorf(codes.Unimplemented, "method PushPaths not implemented")
}
func (UnimplementedPathMessengerServer) mustEmbedUnimplementedPathMessengerServer() {}

// UnsafePathMessengerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PathMessengerServer will
// result in compilation errors.
type UnsafePathMessengerServer interface {
	mustEmbedUnimplementedPathMessengerServer()
}

func RegisterPathMessengerServer(s grpc.ServiceRegistrar, srv PathMessengerServer) {
	s.RegisterService(&_PathMessenger_serviceDesc, srv)
}

func _PathMessenger_StartTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PathMessengerServer).StartTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/messaging.PathMessenger/StartTask",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PathMessengerServer).StartTask(ctx, req.(*TaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PathMessenger_AcceptMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PathMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PathMessengerServer).AcceptMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/messaging.PathMessenger/AcceptMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PathMessengerServer).AcceptMessage(ctx, req.(*PathMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _PathMessenger_GetMessagingData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MessagingDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PathMessengerServer).GetMessagingData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/messaging.PathMessenger/GetMessagingData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PathMessengerServer).GetMessagingData(ctx, req.(*MessagingDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PathMessenger_PushPaths_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(PathMessengerServer).PushPaths(&pathMessengerPushPathsServer{stream})
}

type PathMessenger_PushPathsServer interface {
	SendAndClose(*ConnectionResponse) error
	Recv() (*Edge, error)
	grpc.ServerStream
}

type pathMessengerPushPathsServer struct {
	grpc.ServerStream
}

func (x *pathMessengerPushPathsServer) SendAndClose(m *ConnectionResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *pathMessengerPushPathsServer) Recv() (*Edge, error) {
	m := new(Edge)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _PathMessenger_serviceDesc = grpc.ServiceDesc{
	ServiceName: "messaging.PathMessenger",
	HandlerType: (*PathMessengerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StartTask",
			Handler:    _PathMessenger_StartTask_Handler,
		},
		{
			MethodName: "AcceptMessage",
			Handler:    _PathMessenger_AcceptMessage_Handler,
		},
		{
			MethodName: "GetMessagingData",
			Handler:    _PathMessenger_GetMessagingData_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "PushPaths",
			Handler:       _PathMessenger_PushPaths_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "messaging/shortest_paths.proto",
}
