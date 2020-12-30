package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/nmalensek/shortest-paths/addressing"
	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type registrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu              sync.Mutex //prevent registration overwrites
	registeredNodes map[string]*messaging.Node
}

//RegisterNode stores the address of the sender in a map of known nodes.
func (s *registrationServer) RegisterNode(ctx context.Context, n *messaging.Node) (*messaging.RegistrationResponse, error) {

	return &messaging.RegistrationResponse{}, nil
}

//DeregisterNode removes the node address from the map of known nodes.
func (s *registrationServer) DeregisterNode(ctx context.Context, n *messaging.Node) (*messaging.DeregistrationResponse, error) {
	return &messaging.DeregistrationResponse{}, nil
}

func (s *registrationServer) GetConnections(n *messaging.Node, stream messaging.OverlayRegistration_GetConnectionsServer) error {
	return nil
}

func (s *registrationServer) GetEdges(er *messaging.EdgesRequest, stream messaging.OverlayRegistration_GetEdgesServer) error {
	return nil
}

func (s *registrationServer) ProcessMetadata(ctx context.Context, mmd *messaging.MessagingMetadata) (*messaging.MetadataConfirmation, error) {
	return nil, nil
}

func makeServer() *registrationServer {
	return &registrationServer{registeredNodes: make(map[string]*messaging.Node)}
}

func main() {
	flag.Parse()

	localIP := addressing.GetIP().String()

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%d", localIP, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	messaging.RegisterOverlayRegistrationServer(grpcServer, makeServer())
	grpcServer.Serve(listener)
}
