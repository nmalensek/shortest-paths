package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nmalensek/shortest-paths/addressing"

	"github.com/nmalensek/shortest-paths/messaging"

	"google.golang.org/grpc"
)

var (
	registrationIP   = flag.String("regIp", "127.0.0.1", "The IP address of the registration node")
	registrationPort = flag.Int("regPort", 10000, "The port the registration node is using")
	taskComplete     = false
	numWorkers       = 5
	workChannel      = make(chan messaging.PathMessage, numWorkers)

	helpText = `this is the help text that will list the available commands eventually`
)

type messengerServer struct {
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

func (s *messengerServer) StartTask(context.Context, *messaging.TaskRequest) (*messaging.TaskConfirmation, error) {
	return nil, nil
}

func (s *messengerServer) PushPaths(stream messaging.PathMessenger_PushPathsServer) error {
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
func (s *messengerServer) ProcessMessage(context.Context, *messaging.PathMessage) (*messaging.PathResponse, error) {
	return nil, nil
}

// Transmits metadata about the messages the node has sent and received over the course of the task.
// Technically shouldn't have to worry about locking here because metadata sends should happen at the end
// of the run if everything goes well, but this might help if it's called early (debugging or something's wrong).
func (s *messengerServer) GetMessagingData(context.Context, *messaging.MessagingDataRequest) (*messaging.MessagingMetadata, error) {
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

func newPeerServer() *messengerServer {
	return &messengerServer{nodePathDict: make(map[string]*messaging.Node)}
}

func startPeerService(c chan string) {
	addr := addressing.GetIP()
	go func() {
		listen, err := net.Listen("tcp", fmt.Sprintf("%v:0", addr.String()))
		if err != nil {
			c <- "error: failed to start the peer service"
		}
		c <- listen.Addr().String()

		peerServe := grpc.NewServer()
		messaging.RegisterPathMessengerServer(peerServe, newPeerServer())
		serveErr := peerServe.Serve(listen)
		if serveErr != nil {
			log.Fatalf("server error: %v", serveErr)
		}
	}()
}

func registerToOverlay(c messaging.OverlayRegistrationClient, serveAddr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.RegisterNode(ctx, &messaging.Node{Id: serveAddr})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()

	opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}

	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", *registrationIP, *registrationPort), opts...)
	if err != nil {
		log.Fatalf("failed to connect to server %v:%v", *registrationIP, *registrationPort)
	}
	defer conn.Close()

	// startPeerService starts a GRPC server on a random open port in a goroutine, capture its address
	// and use it to register to the overlay.
	peerChan := make(chan string)
	startPeerService(peerChan)
	myIP := <-peerChan
	if strings.HasPrefix(myIP, "error") {
		log.Fatal(myIP)
	}
	fmt.Printf("Listening on %v\n", myIP)

	regClient := messaging.NewOverlayRegistrationClient(conn)
	err = registerToOverlay(regClient, myIP)
	if err != nil {
		log.Fatalf("registration failed with error %v", err)
	}
	fmt.Printf("Successfully registered to overlay\n")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := scanner.Text()
		switch command {
		case "exit":
			os.Exit(0)
		default:
			fmt.Printf("command %v not recognized, available options are:\n%v\n", command, helpText)
		}
	}
}
