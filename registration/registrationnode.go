package registration

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc"
)

type dialer interface {
	DialFunc(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

type RegistrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu              sync.Mutex
	overlaySent     bool
	registeredNodes map[string]*messaging.Node
	nodeConnections []messaging.PathMessengerClient
	opts            []grpc.DialOption
	dial            func(string, ...grpc.DialOption) (*grpc.ClientConn, error)
}

func New(opts []grpc.DialOption) *RegistrationServer {
	return &RegistrationServer{
		registeredNodes: make(map[string]*messaging.Node),
		dial:            grpc.Dial,
		opts:            opts,
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
	s.registeredNodes[n.GetId()] = n
	conn, err := s.dial(fmt.Sprintf("%v:%v", a.IP, a.Port), s.opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server %v:%v", a.IP, a.Port)
	}
	pClient := messaging.NewPathMessengerClient(conn)
	s.nodeConnections = append(s.nodeConnections, pClient)
	s.mu.Unlock()

	fmt.Printf("registered node: %v\n", n.String())

	return &messaging.RegistrationResponse{}, nil
}

//DeregisterNode removes the node address from the map of known nodes.
func (s *RegistrationServer) DeregisterNode(ctx context.Context, n *messaging.Node) (*messaging.DeregistrationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.overlaySent {
		return &messaging.DeregistrationResponse{}, errors.New("overlay has been sent, cannot deregister node")
	}

	n, ok := s.registeredNodes[n.GetId()]
	if !ok {
		return &messaging.DeregistrationResponse{}, fmt.Errorf("could not find node %v", n.GetId())
	}

	delete(s.registeredNodes, n.Id)

	return &messaging.DeregistrationResponse{}, nil
}

func (s *RegistrationServer) GetOverlay(e *messaging.EdgeRequest, stream messaging.OverlayRegistration_GetOverlayServer) error {
	return nil
}

func (s *RegistrationServer) ProcessMetadata(ctx context.Context, mmd *messaging.MessagingMetadata) (*messaging.MetadataConfirmation, error) {
	return nil, nil
}
