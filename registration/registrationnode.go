package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/nmalensek/shortest-paths/addressing"
	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
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
	err := addressBelongsToSender(ctx, n.GetId())
	if err != nil {
		return nil, err
	}

	_, present := s.registeredNodes[n.GetId()]
	if present {
		return nil, errors.New("address already registered")
	}

	s.mu.Lock()
	s.registeredNodes[n.GetId()] = n
	s.mu.Unlock()

	fmt.Printf("registered node: %v\n", n.String())

	return &messaging.RegistrationResponse{}, nil
}

//DeregisterNode removes the node address from the map of known nodes.
func (s *registrationServer) DeregisterNode(ctx context.Context, n *messaging.Node) (*messaging.DeregistrationResponse, error) {
	err := addressBelongsToSender(ctx, n.GetId())
	if err != nil {
		return nil, err
	}
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

func addressBelongsToSender(ctx context.Context, id string) error {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr.String() != id {
		err := errors.New("could not register node (address mismatch)")
		log.Println(err)
		return err
	}
	return nil
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
