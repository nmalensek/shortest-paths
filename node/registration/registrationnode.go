package registration

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/overlay"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rs/zerolog"
)

type dialer interface {
	DialFunc(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
}

// Config contains all the config variables to run a registration node
type Config struct {
	// The server port
	Port int `json:"port"`

	// The number of message batches every other node in the overlay needs to send
	Batches int `json:"batches"`

	// The number of messages in a batch
	MessagesPerBatch int `json:"messagesPerBatch"`

	// The number of nodes every other node must be connected to
	Connections int `json:"connections"`

	// The total number of nodes that need to be present in the overlay before starting the task
	Peers int `json:"peers"`

	LogLevel string `json:"logLevel"`
}

// RegistrationServer contains everything needed to orchestrate overlay nodes' task(s).
type RegistrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu                  sync.RWMutex
	settings            Config
	logger              zerolog.Logger
	overlaySent         bool
	opts                []grpc.DialOption
	dial                func(string, ...grpc.DialOption) (*grpc.ClientConn, error)
	metadata            map[string]*messaging.MessagingMetadata
	newRegistrationChan chan struct{}

	nodeStatusChan chan *messaging.NodeStatus
	idleNodes      map[string]struct{}
	finishedNodes  map[string]struct{}
	startTime      time.Time

	// Nodes that have registered; used to build the overlay.
	registeredNodes map[string]*messaging.Node

	// List of connected nodes; used to communicate with those nodes.
	nodeConnections map[string]messaging.PathMessengerClient

	overlay []*messaging.Edge
}

// New provides a new instance of RegistrationServer.
func New(opts []grpc.DialOption, conf Config) *RegistrationServer {
	rs := &RegistrationServer{
		registeredNodes:     make(map[string]*messaging.Node),
		dial:                grpc.Dial,
		metadata:            make(map[string]*messaging.MessagingMetadata),
		opts:                opts,
		settings:            conf,
		newRegistrationChan: make(chan struct{}),
		nodeStatusChan:      make(chan *messaging.NodeStatus),
		idleNodes:           make(map[string]struct{}),
		finishedNodes:       make(map[string]struct{}),
		nodeConnections:     make(map[string]messaging.PathMessengerClient),
	}

	logLevel, err := zerolog.ParseLevel(conf.LogLevel)
	if err != nil {
		log.Fatalf("invalid zerolog log level provided (%v)", conf.LogLevel)
	}

	rs.logger = zerolog.New(os.Stdout).With().
		Timestamp().
		Caller().
		Str("node", fmt.Sprintf("registration:%v", conf.Port)).
		Logger().Level(logLevel)

	go rs.monitorOverlayStatus()

	return rs
}

// RegisterNode stores the address of the sender in a map of known nodes.
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
	s.nodeConnections[n.Id] = pClient

	s.logger.Info().Str("successful registration", n.GetId()).Msg("")

	if len(s.registeredNodes) == s.settings.Peers {
		s.newRegistrationChan <- struct{}{}
	}

	return &messaging.RegistrationResponse{}, nil
}

// DeregisterNode removes the node address from the map of known nodes.
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

	s.logger.Info().Msgf("node %v successfully deregistered\n", n.GetId())

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
	// s.overlay = overlay.BuildOverlay(nodes, s.settings.Connections, false)

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

func (s *RegistrationServer) NodeReady(ctx context.Context, n *messaging.NodeStatus) (*messaging.TaskReadyResponse, error) {
	s.nodeStatusChan <- n
	return &messaging.TaskReadyResponse{}, nil
}

func (s *RegistrationServer) NodeFinished(ctx context.Context, n *messaging.NodeStatus) (*messaging.TaskCompleteResponse, error) {
	s.nodeStatusChan <- n
	return &messaging.TaskCompleteResponse{}, nil
}

func (s *RegistrationServer) monitorOverlayStatus() {
	for {
		select {
		case <-s.newRegistrationChan:
			// TODO: do this from a cli client
			errs := s.buildAndPushOverlay()
			if len(errs) > 0 {
				s.logger.Error().Errs("task construction errors", errs).Msg("")
			}
		case n := <-s.nodeStatusChan:
			s.mu.Lock()
			switch {
			case n.Status == messaging.NodeStatus_READY:
				s.idleNodes[n.Id] = struct{}{}
				if len(s.idleNodes) == s.settings.Peers {
					s.logger.Info().Msgf("%v nodes ready, starting task...", len(s.idleNodes))
					go s.startTasks()
				}
			case n.Status == messaging.NodeStatus_COMPLETE:
				s.logger.Info().Str("node finished", n.Id).Msg("")
				s.finishedNodes[n.Id] = struct{}{}
				if len(s.finishedNodes) == len(s.registeredNodes) {
					go s.requestMetadata()
				}
			}
			s.mu.Unlock()
		}

	}
}

func (s *RegistrationServer) startTasks() {
	for k, n := range s.nodeConnections {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
		_, err := n.StartTask(ctx, &messaging.TaskRequest{
			BatchesToSend:    int64(s.settings.Batches),
			MessagesPerBatch: int64(s.settings.MessagesPerBatch),
		})
		if err == nil {
			s.logger.Info().Str("node starting task", k).Msg("")
		}
		cancel()

		retries := 0
		for err != nil && retries < 3 {
			time.Sleep(time.Millisecond * 100)
			_, err = n.StartTask(ctx, &messaging.TaskRequest{
				BatchesToSend:    int64(s.settings.Batches),
				MessagesPerBatch: int64(s.settings.MessagesPerBatch),
			})
			retries++
			if retries == 3 {
				s.logger.Error().Msgf("unable to successfully tell node %v to start task processing after %v retries, skipping...\n", k, retries)
			}
		}
	}

	s.startTime = time.Now()
}

func (s *RegistrationServer) requestMetadata() {
	for k := range s.finishedNodes {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
		data, err := s.nodeConnections[k].GetMessagingData(ctx, &messaging.MessagingDataRequest{})
		if err != nil {
			s.logger.Error().Msgf("failed to get metadata from node %v: %v", k, err)
		}
		s.metadata[k] = data

		cancel()
	}

	printMetadata(s.metadata, s.startTime)
}

// ProcessMetadata prints out formatted metadata about the most recently completed task.
func (s *RegistrationServer) ProcessMetadata(ctx context.Context, mmd *messaging.MessagingMetadata) (*messaging.MetadataConfirmation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metadata[mmd.GetSender().Id] = mmd

	if len(s.metadata) == len(s.registeredNodes) {
		printMetadata(s.metadata, s.startTime)
	}

	return &messaging.MetadataConfirmation{}, nil
}

func printMetadata(d map[string]*messaging.MessagingMetadata, start time.Time) {
	var totalMessagesSent int64 = 0
	var totalMessagesReceived int64 = 0
	var totalMessagesRelayed int64 = 0

	var totalPayloadSent int64 = 0
	var totalPayloadReceived int64 = 0

	tw := tabwriter.NewWriter(os.Stdout, 20, 8, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(tw, "Node", "\t", "Messages Sent", "\t", "Messages Received", "\t", "Messages Relayed", "\t", "Payload Sent", "\t", "Payload Received")

	for k, v := range d {
		totalMessagesSent += v.MessagesSent
		totalMessagesReceived += v.MessagesReceived
		totalMessagesRelayed += v.MessagesRelayed

		totalPayloadSent += v.PayloadSent
		totalPayloadReceived += v.PayloadReceived

		fmt.Fprintln(tw, k, "\t", v.GetMessagesSent(), "\t", v.GetMessagesReceived(), "\t", v.GetMessagesRelayed(), "\t", v.GetPayloadSent(), "\t", v.GetPayloadReceived())
	}
	fmt.Fprintln(tw, "---", "\t", "---", "\t", "---", "\t", "---", "\t", "---", "\t", "---")
	fmt.Fprintln(tw, "Total", "\t", totalMessagesSent, "\t", totalMessagesReceived, "\t", totalMessagesRelayed, "\t", totalPayloadSent, "\t", totalPayloadReceived)
	tw.Flush()

	fmt.Printf("Task took %v seconds", time.Since(start).Seconds())
}
