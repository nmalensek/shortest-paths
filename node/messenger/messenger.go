package messenger

import (
	"context"
	"io"
	"sync"

	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MessengerServer is an instance of a messenger (worker) in an overlay
type MessengerServer struct {
	messaging.UnimplementedPathMessengerServer
	mu           sync.Mutex
	taskComplete bool
	workChan     chan messaging.PathMessage
	pathChan     chan struct{}

	nodePathDict map[string]*messaging.Node
	overlayEdges []*messaging.Edge

	totalCount       int64
	messagesSent     int64
	messagesReceived int64
	messagesRelayed  int64
	payloadSent      int64
	payloadReceived  int64
}

// New returns a new instance of MessengerServer.
func New() *MessengerServer {
	ms := &MessengerServer{
		nodePathDict: make(map[string]*messaging.Node),
		pathChan:     make(chan struct{}),
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
	// calculate shortest paths for each node

	// initialize proper number of workers based on overlay size

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
