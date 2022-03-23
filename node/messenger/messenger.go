package messenger

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/overlay"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MessengerServer is an instance of a messenger (worker) in an overlay
type MessengerServer struct {
	messaging.UnimplementedPathMessengerServer
	serverAddress   string
	registratonConn messaging.OverlayRegistrationClient
	mu              sync.Mutex
	pathChan        chan struct{}

	nodePathDict map[string][]string
	overlayEdges []*messaging.Edge
	nodeConns    map[string]messaging.PathMessengerClient

	workChan     chan messaging.PathMessage
	maxWorkers   int
	sem          *semaphore.Weighted
	taskComplete bool

	messagesSentRequirement int64
	batchMessages           int
	totalCount              int64
	messagesSent            int64
	messagesReceived        int64
	messagesRelayed         int64
	payloadSent             int64
	payloadReceived         int64
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
func (s *MessengerServer) StartTask(ctx context.Context, tr *messaging.TaskRequest) (*messaging.TaskConfirmation, error) {
	s.messagesSentRequirement = tr.BatchesToSend * tr.MessagesPerBatch
	s.batchMessages = int(tr.MessagesPerBatch)

	if s.messagesSentRequirement < 0 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("messages times batches to send must be positive"))
	}

	go s.doTask()

	return &messaging.TaskConfirmation{}, nil
}

func (s *MessengerServer) doTask() {
	// make list of node IDs to randomly select one
	addrList := make([]string, 0, len(s.nodePathDict))
	for addr := range s.nodePathDict {
		addrList = append(addrList, addr)
	}

	randomGenerator := rand.New(rand.NewSource(time.Now().Unix()))

	for s.messagesSent < s.messagesSentRequirement {
		// choose random recipient from connection list
		r := addrList[randomGenerator.Intn(len(addrList))]

		// send to first node on shortest path to dest
		firstNodeConn := s.nodeConns[s.nodePathDict[r][0]]

		for i := 0; i < s.batchMessages; i++ {
			// random signed int32 between MinInt32 and MaxInt32
			p := int32(randomGenerator.Int63n(math.MaxInt32-math.MinInt32) + math.MinInt32)

			// create message with recipient as destination and random payload
			msg := &messaging.PathMessage{
				Payload: p,
				Destination: &messaging.Node{
					Id: r,
				},
				Path: []*messaging.Node{
					{
						Id: s.serverAddress,
					},
				},
			}

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
			_, err := firstNodeConn.AcceptMessage(ctx, msg)
			cancel()

			if err != nil {
				sleepTime := 5
				fmt.Printf("error sending a message to %v, trying again in %v ms: %v", s.nodePathDict[r][0], sleepTime, err)
				time.Sleep(time.Millisecond * time.Duration(sleepTime))
				continue
			}

			s.messagesSent++
			s.payloadSent += int64(p)
		}
	}
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
		log.Fatalf("failed to get shortest paths for node %v: %v", s.serverAddress, err)
	}

	s.nodePathDict = paths

	// connect to the first node in each path and store connection in dict (used by task goroutine)
	for addr := range s.nodePathDict {
		opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}

		conn, err := grpc.Dial(addr, opts...)
		if err != nil {
			log.Fatalf("failed to connect to node %v", addr)
		}
		n := messaging.NewPathMessengerClient(conn)
		s.nodeConns[addr] = n
	}

	s.maxWorkers = len(otherNodes)
	s.workChan = make(chan messaging.PathMessage, len(otherNodes)*5)
	s.sem = semaphore.NewWeighted(int64(s.maxWorkers))

	// tell registration node this node's ready
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
	_, err = s.registratonConn.NodeReady(ctx, &messaging.Node{Id: s.serverAddress})
	if err != nil {
		log.Fatalf("failed to notify registration node that this node is ready")
	}
	defer cancel()

	fmt.Println("Successfully notified registration node that this node is ready")
}

// AcceptMessage either relays the message another hop toward its destination or processes the payload value if the node is the destination.
func (s *MessengerServer) AcceptMessage(context.Context, *messaging.PathMessage) (*messaging.PathResponse, error) {
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
