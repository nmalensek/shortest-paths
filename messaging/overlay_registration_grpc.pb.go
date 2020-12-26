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

// OverlayRegistrationClient is the client API for OverlayRegistration service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type OverlayRegistrationClient interface {
	//Provides node information (host, port) to register the node as part of the overlay. The host and port are the address information for the node's own gRPC service to pass messages between overlay nodes.
	RegisterNode(ctx context.Context, in *Node, opts ...grpc.CallOption) (*RegistrationResponse, error)
	//Provides node information to leave the overlay after registering but before any connections have been established.
	DeregisterNode(ctx context.Context, in *Node, opts ...grpc.CallOption) (*DeregistrationResponse, error)
	//Sends a stream of Nodees that the calling node should connect to.
	GetConnections(ctx context.Context, in *Node, opts ...grpc.CallOption) (OverlayRegistration_GetConnectionsClient, error)
	//Sends a stream of Edges that represent the entire overlay so each node can calculate its shortest paths from its established connections.
	GetEdges(ctx context.Context, in *EdgesRequest, opts ...grpc.CallOption) (OverlayRegistration_GetEdgesClient, error)
	//Allows overlay nodes to transmit their metadata when it's ready (i.e., when done with a task, etc.)
	ProcessMetadata(ctx context.Context, in *MessagingMetadata, opts ...grpc.CallOption) (*MetadataConfirmation, error)
}

type overlayRegistrationClient struct {
	cc grpc.ClientConnInterface
}

func NewOverlayRegistrationClient(cc grpc.ClientConnInterface) OverlayRegistrationClient {
	return &overlayRegistrationClient{cc}
}

func (c *overlayRegistrationClient) RegisterNode(ctx context.Context, in *Node, opts ...grpc.CallOption) (*RegistrationResponse, error) {
	out := new(RegistrationResponse)
	err := c.cc.Invoke(ctx, "/messaging.OverlayRegistration/RegisterNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *overlayRegistrationClient) DeregisterNode(ctx context.Context, in *Node, opts ...grpc.CallOption) (*DeregistrationResponse, error) {
	out := new(DeregistrationResponse)
	err := c.cc.Invoke(ctx, "/messaging.OverlayRegistration/DeregisterNode", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *overlayRegistrationClient) GetConnections(ctx context.Context, in *Node, opts ...grpc.CallOption) (OverlayRegistration_GetConnectionsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_OverlayRegistration_serviceDesc.Streams[0], "/messaging.OverlayRegistration/GetConnections", opts...)
	if err != nil {
		return nil, err
	}
	x := &overlayRegistrationGetConnectionsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type OverlayRegistration_GetConnectionsClient interface {
	Recv() (*Node, error)
	grpc.ClientStream
}

type overlayRegistrationGetConnectionsClient struct {
	grpc.ClientStream
}

func (x *overlayRegistrationGetConnectionsClient) Recv() (*Node, error) {
	m := new(Node)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *overlayRegistrationClient) GetEdges(ctx context.Context, in *EdgesRequest, opts ...grpc.CallOption) (OverlayRegistration_GetEdgesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_OverlayRegistration_serviceDesc.Streams[1], "/messaging.OverlayRegistration/GetEdges", opts...)
	if err != nil {
		return nil, err
	}
	x := &overlayRegistrationGetEdgesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type OverlayRegistration_GetEdgesClient interface {
	Recv() (*Edge, error)
	grpc.ClientStream
}

type overlayRegistrationGetEdgesClient struct {
	grpc.ClientStream
}

func (x *overlayRegistrationGetEdgesClient) Recv() (*Edge, error) {
	m := new(Edge)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *overlayRegistrationClient) ProcessMetadata(ctx context.Context, in *MessagingMetadata, opts ...grpc.CallOption) (*MetadataConfirmation, error) {
	out := new(MetadataConfirmation)
	err := c.cc.Invoke(ctx, "/messaging.OverlayRegistration/ProcessMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OverlayRegistrationServer is the server API for OverlayRegistration service.
// All implementations must embed UnimplementedOverlayRegistrationServer
// for forward compatibility
type OverlayRegistrationServer interface {
	//Provides node information (host, port) to register the node as part of the overlay. The host and port are the address information for the node's own gRPC service to pass messages between overlay nodes.
	RegisterNode(context.Context, *Node) (*RegistrationResponse, error)
	//Provides node information to leave the overlay after registering but before any connections have been established.
	DeregisterNode(context.Context, *Node) (*DeregistrationResponse, error)
	//Sends a stream of Nodees that the calling node should connect to.
	GetConnections(*Node, OverlayRegistration_GetConnectionsServer) error
	//Sends a stream of Edges that represent the entire overlay so each node can calculate its shortest paths from its established connections.
	GetEdges(*EdgesRequest, OverlayRegistration_GetEdgesServer) error
	//Allows overlay nodes to transmit their metadata when it's ready (i.e., when done with a task, etc.)
	ProcessMetadata(context.Context, *MessagingMetadata) (*MetadataConfirmation, error)
	mustEmbedUnimplementedOverlayRegistrationServer()
}

// UnimplementedOverlayRegistrationServer must be embedded to have forward compatible implementations.
type UnimplementedOverlayRegistrationServer struct {
}

func (UnimplementedOverlayRegistrationServer) RegisterNode(context.Context, *Node) (*RegistrationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterNode not implemented")
}
func (UnimplementedOverlayRegistrationServer) DeregisterNode(context.Context, *Node) (*DeregistrationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeregisterNode not implemented")
}
func (UnimplementedOverlayRegistrationServer) GetConnections(*Node, OverlayRegistration_GetConnectionsServer) error {
	return status.Errorf(codes.Unimplemented, "method GetConnections not implemented")
}
func (UnimplementedOverlayRegistrationServer) GetEdges(*EdgesRequest, OverlayRegistration_GetEdgesServer) error {
	return status.Errorf(codes.Unimplemented, "method GetEdges not implemented")
}
func (UnimplementedOverlayRegistrationServer) ProcessMetadata(context.Context, *MessagingMetadata) (*MetadataConfirmation, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProcessMetadata not implemented")
}
func (UnimplementedOverlayRegistrationServer) mustEmbedUnimplementedOverlayRegistrationServer() {}

// UnsafeOverlayRegistrationServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to OverlayRegistrationServer will
// result in compilation errors.
type UnsafeOverlayRegistrationServer interface {
	mustEmbedUnimplementedOverlayRegistrationServer()
}

func RegisterOverlayRegistrationServer(s grpc.ServiceRegistrar, srv OverlayRegistrationServer) {
	s.RegisterService(&_OverlayRegistration_serviceDesc, srv)
}

func _OverlayRegistration_RegisterNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Node)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OverlayRegistrationServer).RegisterNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/messaging.OverlayRegistration/RegisterNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OverlayRegistrationServer).RegisterNode(ctx, req.(*Node))
	}
	return interceptor(ctx, in, info, handler)
}

func _OverlayRegistration_DeregisterNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Node)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OverlayRegistrationServer).DeregisterNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/messaging.OverlayRegistration/DeregisterNode",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OverlayRegistrationServer).DeregisterNode(ctx, req.(*Node))
	}
	return interceptor(ctx, in, info, handler)
}

func _OverlayRegistration_GetConnections_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Node)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(OverlayRegistrationServer).GetConnections(m, &overlayRegistrationGetConnectionsServer{stream})
}

type OverlayRegistration_GetConnectionsServer interface {
	Send(*Node) error
	grpc.ServerStream
}

type overlayRegistrationGetConnectionsServer struct {
	grpc.ServerStream
}

func (x *overlayRegistrationGetConnectionsServer) Send(m *Node) error {
	return x.ServerStream.SendMsg(m)
}

func _OverlayRegistration_GetEdges_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(EdgesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(OverlayRegistrationServer).GetEdges(m, &overlayRegistrationGetEdgesServer{stream})
}

type OverlayRegistration_GetEdgesServer interface {
	Send(*Edge) error
	grpc.ServerStream
}

type overlayRegistrationGetEdgesServer struct {
	grpc.ServerStream
}

func (x *overlayRegistrationGetEdgesServer) Send(m *Edge) error {
	return x.ServerStream.SendMsg(m)
}

func _OverlayRegistration_ProcessMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MessagingMetadata)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OverlayRegistrationServer).ProcessMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/messaging.OverlayRegistration/ProcessMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OverlayRegistrationServer).ProcessMetadata(ctx, req.(*MessagingMetadata))
	}
	return interceptor(ctx, in, info, handler)
}

var _OverlayRegistration_serviceDesc = grpc.ServiceDesc{
	ServiceName: "messaging.OverlayRegistration",
	HandlerType: (*OverlayRegistrationServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterNode",
			Handler:    _OverlayRegistration_RegisterNode_Handler,
		},
		{
			MethodName: "DeregisterNode",
			Handler:    _OverlayRegistration_DeregisterNode_Handler,
		},
		{
			MethodName: "ProcessMetadata",
			Handler:    _OverlayRegistration_ProcessMetadata_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetConnections",
			Handler:       _OverlayRegistration_GetConnections_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetEdges",
			Handler:       _OverlayRegistration_GetEdges_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "overlay_registration.proto",
}
