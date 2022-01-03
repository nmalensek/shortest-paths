package messenger

import (
	"context"
	"io"
	"log"
	"sync"

	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/overlay"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MessengerServer is an instance of a messenger (worker) in an overlay
type MessengerServer struct {
	messaging.UnimplementedPathMessengerServer
	serverAddress string
	mu            sync.Mutex
	taskComplete  bool
	workChan      chan messaging.PathMessage
	pathChan      chan struct{}

	nodePathDict map[string][]string
	overlayEdges []*messaging.Edge

	totalCount       int64
	messagesSent     int64
	messagesReceived int64
	messagesRelayed  int64
	payloadSent      int64
	payloadReceived  int64
}

// New returns a new instance of MessengerServer.
func New(serverAddr string) *MessengerServer {
	ms := &MessengerServer{
		serverAddress: serverAddr,
		nodePathDict:  make(map[string][]string),
		pathChan:      make(chan struct{}),
	}
	go ms.calculatePathsWhenReady()

	return ms
}

// StartTask starts the messenger's task.
func (s *MessengerServer) StartTask(context.Context, *messaging.TaskRequest) (*messaging.TaskConfirmation, error) {
	return nil, nil
}

// PushPaths receives the stream of edges that make up the overlay the node is part of.
func (s *MessengerServer) PushPaths(stream messaging.PathMessenger_PushPathsServer) error {
	if stream.Context().Err() == context.Canceled {
		return status.Error(codes.Canceled, "sender canceled path push, aborting...")
	}

	for {
		edge, err := stream.Recv()
		if err == io.EOF {
			s.pathChan <- struct{}{}
			return stream.SendAndClose(&messaging.ConnectionResponse{})
		}
		if err != nil {
			return err
		}

		s.overlayEdges = append(s.overlayEdges, edge)
	}
}

func (s *MessengerServer) calculatePathsWhenReady() {
	<-s.pathChan
	// map of each node to edges it's part of.
	overlayConnections := make(map[string][]*messaging.Edge)
	// keep track of all other nodes as nodes this one should connect to.
	otherNodes := make(map[string]struct{})

	for _, e := range s.overlayEdges {
		if e.Source.Id != s.serverAddress {
			otherNodes[e.Source.Id] = struct{}{}
		}
		if e.Destination.Id != s.serverAddress {
			otherNodes[e.Destination.Id] = struct{}{}
		}

		overlayConnections[e.Source.Id] = append(overlayConnections[e.Source.Id], e)
		overlayConnections[e.Destination.Id] = append(overlayConnections[e.Destination.Id], e)
	}

	paths, err := overlay.GetAllShortestPaths(s.serverAddress, otherNodes, overlayConnections)
	if err != nil {
		// TODO: send to registration node
		log.Fatalf("failed to get shortest paths for node %v: %v", s.serverAddress, err)
	}

	s.nodePathDict = paths

	// initialize proper number of workers based on overlay size (use a semaphore reading from a work channel)

	// tell registration node this node's ready

}

// ProcessMessage either relays the message another hop toward its destination or processes the payload value if the node is the destination.
func (s *MessengerServer) ProcessMessage(context.Context, *messaging.PathMessage) (*messaging.PathResponse, error) {
	return nil, nil
}

// GetMessagingData transmits metadata about the messages the node has sent and received over the course of the task.
func (s *MessengerServer) GetMessagingData(context.Context, *messaging.MessagingDataRequest) (*messaging.MessagingMetadata, error) {
	// Technically shouldn't have to worry about locking here because metadata sends should happen at the end
	// of the run if everything goes well, but this might help if it's called early (debugging or something's wrong).
	s.mu.Lock()
	data := &messaging.MessagingMetadata{
		MessagesSent:     s.messagesSent,
		MessagesReceived: s.messagesReceived,
		MessagesRelayed:  s.messagesRelayed,
		PayloadSent:      s.payloadSent,
		PayloadReceived:  s.payloadReceived,
	}
	s.mu.Unlock()

	return data, nil
}
