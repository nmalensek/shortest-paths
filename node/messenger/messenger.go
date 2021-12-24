package messenger

import (
	"context"
	"io"
	"sync"

	"github.com/nmalensek/shortest-paths/messaging"
)

// MessengerServer is an instance of a messenger (worker) in an overlay
type MessengerServer struct {
	messaging.UnimplementedPathMessengerServer
	mu               sync.Mutex
	totalCount       int64
	messagesSent     int64
	messagesReceived int64
	messagesRelayed  int64
	payloadSent      int64
	payloadReceived  int64
	nodePathDict     map[string]*messaging.Node
	overlayEdges     []*messaging.Edge
}

// New returns a new instance of MessengerServer
func New() *MessengerServer {
	return &MessengerServer{nodePathDict: make(map[string]*messaging.Node)}
}

func (s *MessengerServer) StartTask(context.Context, *messaging.TaskRequest) (*messaging.TaskConfirmation, error) {
	return nil, nil
}

func (s *MessengerServer) PushPaths(stream messaging.PathMessenger_PushPathsServer) error {
	for {
		edge, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&messaging.ConnectionResponse{})
		}
		if err != nil {
			return err
		}

		s.overlayEdges = append(s.overlayEdges, edge)
	}
}

// Either relays the message another hop toward its destination or processes the payload value if the node is the destination.
func (s *MessengerServer) ProcessMessage(context.Context, *messaging.PathMessage) (*messaging.PathResponse, error) {
	return nil, nil
}

// Transmits metadata about the messages the node has sent and received over the course of the task.
// Technically shouldn't have to worry about locking here because metadata sends should happen at the end
// of the run if everything goes well, but this might help if it's called early (debugging or something's wrong).
func (s *MessengerServer) GetMessagingData(context.Context, *messaging.MessagingDataRequest) (*messaging.MessagingMetadata, error) {
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
