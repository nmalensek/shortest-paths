package registration

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nmalensek/shortest-paths/config"
	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/overlay"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dialer interface {
	DialFunc(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

// RegistrationServer contains everything needed to orchestrate overlay nodes' task(s).
type RegistrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu                  sync.RWMutex
	settings            config.RegistrationServer
	overlaySent         bool
	opts                []grpc.DialOption
	dial                func(string, ...grpc.DialOption) (*grpc.ClientConn, error)
	metadata            map[string]*messaging.MessagingMetadata
	newRegistrationChan chan struct{}

	nodeStatusChan chan struct{}
	idleNodes      map[string]struct{}

	// Nodes that have registered; used to build the overlay.
	registeredNodes map[string]*messaging.Node

	// List of connected nodes; used to communicate with those nodes.
	nodeConnections []messaging.PathMessengerClient

	overlay []*messaging.Edge
}

// New provides a new instance of RegistrationServer.
func New(opts []grpc.DialOption, conf config.RegistrationServer) *RegistrationServer {
	rs := &RegistrationServer{
		registeredNodes:     make(map[string]*messaging.Node),
		dial:                grpc.Dial,
		opts:                opts,
		settings:            conf,
		newRegistrationChan: make(chan struct{}),
		nodeStatusChan:      make(chan struct{}),
	}
	go rs.monitorOverlayStatus()

	return rs
}

//RegisterNode stores the address of the sender in a map of known nodes.
func (s *RegistrationServer) RegisterNode(ctx context.Context, n *messaging.Node) (*messaging.RegistrationResponse, error) {
	if ctx.Err() == context.Canceled {
		return &messaging.RegistrationResponse{}, status.Error(codes.Canceled, "client canceled registration, aborting")
	}

	_, present := s.registeredNodes[n.GetId()]
	if present {
		return nil, status.Error(codes.AlreadyExists, "address already registered")
	}

	addr := strings.Split(n.GetId(), ":")
	if len(addr) != 2 {
		return nil, status.Error(codes.InvalidArgument, "node ID must be in the format host:port")
	}

	a, err := net.ResolveTCPAddr("tcp", n.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "TCP address not formatted properly")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.registeredNodes[n.GetId()] = n
	conn, err := s.dial(fmt.Sprintf("%v:%v", a.IP, a.Port), s.opts...)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to connect to %v:%v, ensure server is listening on that port",
			a.IP, a.Port)
	}
	pClient := messaging.NewPathMessengerClient(conn)
	s.nodeConnections = append(s.nodeConnections, pClient)

	fmt.Printf("registered node: %v\n", n.String())

	if len(s.registeredNodes) == s.settings.Peers {
		s.newRegistrationChan <- struct{}{}
	}

	return &messaging.RegistrationResponse{}, nil
}

//DeregisterNode removes the node address from the map of known nodes.
func (s *RegistrationServer) DeregisterNode(ctx context.Context, n *messaging.Node) (*messaging.DeregistrationResponse, error) {
	if ctx.Err() == context.Canceled {
		return &messaging.DeregistrationResponse{}, status.Error(codes.Canceled, "client canceled de-registration, abandoning")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.overlaySent {
		// TODO: allow deregistration; update the shortest paths and re-send.
		return &messaging.DeregistrationResponse{}, status.Errorf(codes.FailedPrecondition,
			"overlay has been sent, cannot deregister node")
	}

	_, ok := s.registeredNodes[n.GetId()]
	if !ok {
		return &messaging.DeregistrationResponse{}, status.Errorf(codes.NotFound, "could not find node %v", n.GetId())
	}

	delete(s.registeredNodes, n.GetId())

	fmt.Printf("node %v successfully deregistered\n", n.GetId())

	return &messaging.DeregistrationResponse{}, nil
}

// GetOverlay returns the current state of the overlay as a stream of edges.
func (s *RegistrationServer) GetOverlay(e *messaging.EdgeRequest, stream messaging.OverlayRegistration_GetOverlayServer) error {
	if !s.overlaySent {
		return status.Error(codes.FailedPrecondition, "overlay has not been set up yet")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, edge := range s.overlay {
		if err := stream.Send(edge); err != nil {
			return err
		}
	}

	return nil
}

func (s *RegistrationServer) buildAndPushOverlay() []error {
	errs := make([]error, 0)

	// build the overlay
	nodes := make([]*messaging.Node, 0, len(s.registeredNodes))
	for _, n := range s.registeredNodes {
		nodes = append(nodes, n)
	}

	s.overlay = overlay.BuildOverlay(nodes, s.settings.Connections, true)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*15))
	defer cancel()

	// push to edges
	for _, n := range s.nodeConnections {
		stream, err := n.PushPaths(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%v.PushPaths failed: %v", n, err))
			continue
		}
		for _, e := range s.overlay {
			if err := stream.Send(e); err != nil {
				errs = append(errs, fmt.Errorf("%v.Send(%v) failed: %v\n", stream, e, err))
				continue
			}
		}
		_, err = stream.CloseAndRecv()
		if err != nil {
			errs = append(errs, fmt.Errorf("%v.CloseAndRecv() failed: %v\n", stream, err))
			continue
		}
	}

	return errs
}

func (s *RegistrationServer) monitorOverlayStatus() {
	for {
		select {
		case <-s.newRegistrationChan:
			// TODO: do this from a cli client
			errs := s.buildAndPushOverlay()
			if len(errs) > 0 {
				fmt.Println("task construction failed with the following errors:")
				for _, e := range errs {
					fmt.Println(e)
				}
			}
		case <-s.nodeStatusChan:
			// new message type: ready/finished
			// if ready, add to list of idle nodes. if that list == overlay size, start task
			// clear list
			// if finished, add node to list of idle nodes. if that list == overlay size, request metadata
		}

	}

}

// ProcessMetadata prints out formatted metadata about the most recently completed task.
func (s *RegistrationServer) ProcessMetadata(ctx context.Context, mmd *messaging.MessagingMetadata) (*messaging.MetadataConfirmation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.metadata[mmd.GetSender().Id] = mmd

	if len(s.metadata) == len(s.registeredNodes) {
		printMetadata(s.metadata)
	}

	return &messaging.MetadataConfirmation{}, nil
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
