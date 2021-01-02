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
	port        = flag.Int("port", 10000, "The server port")
	rounds      = flag.Int("rounds", 1000, "The number of 5-message batches every other node in the overlay needs to send")
	connections = flag.Int("connections", 4, "The number of nodes every other node must be connected to")
	peers       = flag.Int("peers", 10, "The total number of nodes that need to be present in the overlay before starting the task")
)

type registrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu              sync.Mutex
	registeredNodes map[string]*messaging.Node
}

//RegisterNode stores the address of the sender in a map of known nodes.
func (s *registrationServer) RegisterNode(ctx context.Context, n *messaging.Node) (*messaging.RegistrationResponse, error) {
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

//Retrieves the sender's IP address and compares it with the given ID. Intended to ensure a node could only register/deregister itself; however, it's more accurate to do this on the basis of auth tokens or something similar because a node's sending and receiving addresses will be different.
func addressBelongsToSender(ctx context.Context, id string) error {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr.String() != id {
		err := errors.New("could not deregister node (address mismatch)")
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
	fmt.Printf("Listening on %v:%d\n", localIP, *port)

	grpcServer := grpc.NewServer()

	messaging.RegisterOverlayRegistrationServer(grpcServer, makeServer())
	grpcServer.Serve(listener)
}
