package messenger

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/overlay"
	"github.com/rs/zerolog"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Config contains all the configuration needed to register the messenger with the registration node
type Config struct {
	RegistrationIP   string `json:"registrationIP"`
	RegistrationPort int    `json:"registrationPort"`
	LogLevel         string `json:"logLevel"`
}

// MessengerServer is an instance of a messenger (worker) in an overlay
type MessengerServer struct {
	messaging.UnimplementedPathMessengerServer
	serverAddress   string
	registratonConn messaging.OverlayRegistrationClient
	logger          zerolog.Logger
	startTaskChan   chan struct{}
	mu              sync.Mutex
	pathChan        chan struct{}

	nodePathDict map[string][]string
	overlayEdges []*messaging.Edge
	nodeConns    map[string]messaging.PathMessengerClient

	workChan     chan *messaging.PathMessage
	maxWorkers   int
	sem          *semaphore.Weighted
	taskComplete bool

	statsChan               chan recStats
	messagesSentRequirement int64
	batchMessages           int64
	totalCount              int64
	messagesSent            int64
	messagesReceived        int64
	messagesRelayed         int64
	payloadSent             int64
	payloadReceived         int64

	shutdownChan chan struct{}
}

type recStats struct {
	MessageType statsType
	Payload     int32
}

type statsType int

const (
	RECEIVED statsType = iota
	RELAYED
)

// New returns a new instance of MessengerServer.
func New(serverAddr string, regClient messaging.OverlayRegistrationClient, logLevel string) *MessengerServer {
	ms := &MessengerServer{
		serverAddress:   serverAddr,
		registratonConn: regClient,
		nodePathDict:    make(map[string][]string),
		nodeConns:       make(map[string]messaging.PathMessengerClient),
		overlayEdges:    make([]*messaging.Edge, 0, 1),
		pathChan:        make(chan struct{}),
		startTaskChan:   make(chan struct{}),
		statsChan:       make(chan recStats, 1000),
		shutdownChan:    make(chan struct{}),
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("invalid zerolog log level provided (%v)", logLevel)
	}

	ms.logger = zerolog.New(os.Stdout).With().
		Timestamp().
		Caller().
		Str("node", ms.serverAddress).
		Logger().Level(level)

	go ms.calculatePathsWhenReady()
	go ms.doTask()
	go ms.trackReceivedData()

	return ms
}

func (s *MessengerServer) setWorkValues(maxWorkers int) {
	s.maxWorkers = maxWorkers
	s.workChan = make(chan *messaging.PathMessage, maxWorkers*5)
	s.sem = semaphore.NewWeighted(int64(maxWorkers))
}

// StartTask starts the messenger's task.
func (s *MessengerServer) StartTask(ctx context.Context, tr *messaging.TaskRequest) (*messaging.TaskConfirmation, error) {
	s.messagesSentRequirement = tr.BatchesToSend * tr.MessagesPerBatch
	s.batchMessages = tr.MessagesPerBatch

	if s.messagesSentRequirement < 0 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("messages times batches to send must be positive"))
	}

	s.startTaskChan <- struct{}{}

	return &messaging.TaskConfirmation{}, nil
}

func (s *MessengerServer) doTask() {
	<-s.startTaskChan

	if s.batchMessages <= 0 {
		s.logger.Panic().Msg("cannot do task, batchMessages is less than zero")
		return
	}

	// make list of node IDs to randomly select one
	addrList := make([]string, 0, len(s.nodePathDict))
	for addr := range s.nodePathDict {
		addrList = append(addrList, addr)
	}

	randomGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))

	for s.messagesSent < s.messagesSentRequirement {
		// choose random recipient from connection list
		r := addrList[randomGenerator.Intn(len(addrList))]

		// send to first node on shortest path to dest
		firstNodeConn := s.nodeConns[s.nodePathDict[r][0]]

		for i := int64(0); i < s.batchMessages; i++ {
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

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*5000))
			_, err := firstNodeConn.AcceptMessage(ctx, msg)
			cancel()

			if err != nil {
				sleepTime := 1000
				s.logger.Warn().Err(err).Str("destination", s.nodePathDict[r][0]).Int("wait (ms)", sleepTime).Msg("")

				time.Sleep(time.Millisecond * time.Duration(sleepTime))
				continue
			}

			s.messagesSent++
			s.payloadSent += int64(p)
		}
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
	_, err := s.registratonConn.NodeFinished(ctx, &messaging.NodeStatus{Id: s.serverAddress, Status: messaging.NodeStatus_COMPLETE})
	cancel()

	retries := 0
	for err != nil && retries < 3 {
		time.Sleep(time.Millisecond * 100)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
		_, err = s.registratonConn.NodeFinished(ctx, &messaging.NodeStatus{Id: s.serverAddress, Status: messaging.NodeStatus_COMPLETE})
		cancel()
		retries++
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
			s.logger.Debug().Msgf("got overlay: %v", s.overlayEdges)
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
		s.logger.Debug().Str("connected", addr)
	}

	s.setWorkValues(len(otherNodes))
	go s.processMessages()

	// tell registration node this node's ready
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*100))
	_, err = s.registratonConn.NodeReady(ctx, &messaging.NodeStatus{Id: s.serverAddress, Status: messaging.NodeStatus_READY})
	if err != nil {
		log.Fatalf("failed to notify registration node that this node is ready")
	}
	defer cancel()

	s.logger.Debug().Msg("successfully notified registration node that this node is ready")

}

// AcceptMessage either relays the message another hop toward its destination or processes the payload value if the node is the destination.
func (s *MessengerServer) AcceptMessage(ctx context.Context, mp *messaging.PathMessage) (*messaging.PathResponse, error) {
	s.workChan <- mp
	return &messaging.PathResponse{}, nil
}

func (s *MessengerServer) processMessages() {
	if s.workChan == nil || s.shutdownChan == nil || s.statsChan == nil {
		log.Fatal("all work channels must be non-nil")
		return
	}
	for {
		select {
		case m := <-s.workChan:
			if err := s.sem.Acquire(context.TODO(), 1); err != nil {
				s.logger.Warn().Err(err).Msg("failed to acquire semaphore")
				break
			}

			go func() {
				defer s.sem.Release(1)

				if m.Destination == nil {
					s.logger.Error().Str("event", "message with no destination received").Str("message", fmt.Sprintf("%+v", m)).Msg("")
					return
				}

				if m.Destination.Id == s.serverAddress {
					s.logger.Debug().Str("event", "message received").Str("message", fmt.Sprintf("%+v", m)).Msg("")

					s.statsChan <- recStats{
						MessageType: RECEIVED,
						Payload:     m.Payload,
					}
					return
				}

				m.Path = append(m.Path, &messaging.Node{Id: s.serverAddress})
				if len(s.nodePathDict[m.Destination.Id]) == 0 {
					s.logger.Error().Str("destination", m.Destination.Id).Msg("no known path, discarding message")
					return
				}

				nextNode := s.nodePathDict[m.Destination.Id][0]

				s.statsChan <- recStats{
					MessageType: RELAYED,
				}

				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*5000))
				_, err := s.nodeConns[nextNode].AcceptMessage(ctx, m)
				if err != nil {
					s.logger.Err(err).Msgf("%v", m)
				}

				cancel()
			}()
		case <-s.shutdownChan:
			s.logger.Info().Str("event", "processMessages() shutdown signal received").Msg("")
			s.sem.Acquire(context.Background(), int64(s.maxWorkers))
			return
		}
	}
}

func (s *MessengerServer) trackReceivedData() {
	for {
		select {
		case m := <-s.statsChan:
			switch m.MessageType {
			case RECEIVED:
				s.messagesReceived++
				s.payloadReceived += int64(m.Payload)
			case RELAYED:
				s.messagesRelayed++
			}
		case <-s.shutdownChan:
			s.logger.Info().Str("event", "trackReceivedData() shutdown signal received").Msg("")
			return
		}
	}
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

func (s *MessengerServer) logNodeState() {
	s.logger.Debug().
		Str("server address", s.serverAddress).
		Str("path dict", fmt.Sprintf("%v", s.nodePathDict)).
		Str("overlay edges", fmt.Sprintf("%v", s.overlayEdges)).
		Str("node connections", fmt.Sprintf("%v", s.nodeConns)).
		Str("work chan", fmt.Sprintf("%+v", s.workChan)).
		Int("max workers", s.maxWorkers).
		Int("messages sent requirement", int(s.messagesSentRequirement)).
		Stack().
		Msg("")
}
