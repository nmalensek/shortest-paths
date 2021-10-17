package registration

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nmalensek/shortest-paths/config"
	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc"
)

type dialer interface {
	DialFunc(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

type RegistrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu          sync.Mutex
	settings    config.RegistrationServer
	overlaySent bool
	opts        []grpc.DialOption
	dial        func(string, ...grpc.DialOption) (*grpc.ClientConn, error)
	metadata    map[string]*messaging.MessagingMetadata

	// Nodes that have registered; used to build the overlay.
	registeredNodes map[string]*messaging.Node

	// List of connected nodes; used to communicate with those nodes.
	nodeConnections []messaging.PathMessengerClient

	overlay []*messaging.Edge
}

func New(opts []grpc.DialOption, conf config.RegistrationServer) *RegistrationServer {
	return &RegistrationServer{
		registeredNodes: make(map[string]*messaging.Node),
		dial:            grpc.Dial,
		opts:            opts,
		settings:        conf,
	}
}

//RegisterNode stores the address of the sender in a map of known nodes.
func (s *RegistrationServer) RegisterNode(ctx context.Context, n *messaging.Node) (*messaging.RegistrationResponse, error) {
	_, present := s.registeredNodes[n.GetId()]
	if present {
		return nil, errors.New("address already registered")
	}

	addr := strings.Split(n.GetId(), ":")
	if len(addr) != 2 {
		return nil, errors.New("node ID must be in the format host:port")
	}

	a, err := net.ResolveTCPAddr("tcp", n.GetId())
	if err != nil {
		return nil, errors.New("TCP address not formatted properly")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.registeredNodes[n.GetId()] = n
	conn, err := s.dial(fmt.Sprintf("%v:%v", a.IP, a.Port), s.opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server %v:%v", a.IP, a.Port)
	}
	pClient := messaging.NewPathMessengerClient(conn)
	s.nodeConnections = append(s.nodeConnections, pClient)

	fmt.Printf("registered node: %v\n", n.String())

	return &messaging.RegistrationResponse{}, nil
}

//DeregisterNode removes the node address from the map of known nodes.
func (s *RegistrationServer) DeregisterNode(ctx context.Context, n *messaging.Node) (*messaging.DeregistrationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.overlaySent {
		// TODO: allow deregistration; update the shortest paths and re-send.
		return &messaging.DeregistrationResponse{}, errors.New("overlay has been sent, cannot deregister node")
	}

	_, ok := s.registeredNodes[n.GetId()]
	if !ok {
		return &messaging.DeregistrationResponse{}, fmt.Errorf("could not find node %v", n.GetId())
	}

	delete(s.registeredNodes, n.GetId())

	return &messaging.DeregistrationResponse{}, nil
}

func (s *RegistrationServer) GetOverlay(e *messaging.EdgeRequest, stream messaging.OverlayRegistration_GetOverlayServer) error {
	if !s.overlaySent {
		return errors.New("overlay has not been set up yet.")
	}

	for _, edge := range s.overlay {
		if err := stream.Send(edge); err != nil {
			return err
		}
	}

	return nil
}

func (s *RegistrationServer) ProcessMetadata(ctx context.Context, mmd *messaging.MessagingMetadata) (*messaging.MetadataConfirmation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metadata[mmd.GetSender().Id] = mmd

	if len(s.metadata) == len(s.registeredNodes) {
		printMetadata(s.metadata)
	}

	return &messaging.MetadataConfirmation{}, nil
}

var weightGenerator = rand.New(rand.NewSource(time.Now().Unix()))

func (s *RegistrationServer) randomWeight() int {
	return weightGenerator.Intn(51)
}

func (s *RegistrationServer) setRandomConnections() {
	// copy the original list, remove nodes once they have the desired number of connections.
	nodesNeedingConns := make([]*messaging.Node, len(s.registeredNodes))
	for _, v := range s.registeredNodes {
		nodesNeedingConns = append(nodesNeedingConns, v)
	}

	// randomly select nodes from the list, excluding the current node getting connections assigned.

	// record those connections as new edges (edges are bidirectional in this overlay)

}

func printMetadata(d map[string]*messaging.MessagingMetadata) {
	var totalMessagesSent int64 = 0
	var totalMessagesReceived int64 = 0
	var totalMessagesRelayed int64 = 0

	var totalPayloadSent int64 = 0
	var totalPayloadReceived int64 = 0

	fmt.Println("Node\tMessages Sent\tMessages Received\tMessages Relayed\tPayload Sent\t Payload Received")
	for k, v := range d {
		totalMessagesSent += v.MessagesSent
		totalMessagesReceived += v.MessagesSent
		totalMessagesRelayed += v.MessagesRelayed

		totalPayloadSent += v.PayloadSent
		totalPayloadReceived += v.PayloadReceived

		fmt.Println(fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v", k, v.GetMessagesSent(), v.GetMessagesReceived(), v.GetMessagesRelayed(), v.GetPayloadSent(), v.GetPayloadReceived()))
	}
	fmt.Println("-------------------------------------------------------------------------")
	fmt.Println(fmt.Sprintf("\t\t\t%v\t%v\t%v\t%v\t%v", totalMessagesSent, totalMessagesReceived, totalMessagesRelayed, totalPayloadSent, totalPayloadReceived))
}
