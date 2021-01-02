package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
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
}

func (s *messengerServer) StartTask(context.Context, *messaging.TaskRequest) (*messaging.TaskConfirmation, error) {
	return nil, nil
}

//Either relays the message another hop toward its destination or processes the payload value if the node is the destination.
func (s *messengerServer) ProcessMessage(context.Context, *messaging.PathMessage) (*messaging.PathResponse, error) {
	return nil, nil
}

//Transmits metadata about the messages the node has sent and received over the course of the task.
func (s *messengerServer) GetMessagingData(context.Context, *messaging.MessagingDataRequest) (*messaging.MessagingMetadata, error) {
	return nil, nil
}

func newPeerServer() *messengerServer {
	return &messengerServer{nodePathDict: make(map[string]*messaging.Node)}
}

func startPeerService(c chan string) {
	addr := addressing.GetIP()
	go func() {
		listen, err := net.Listen("tcp", fmt.Sprintf("%v:0", addr.String()))
		if err != nil {
			//TODO: make this a separate channel?
			c <- "error: failed to start the peer service"
		}
		c <- listen.Addr().String()

		peerServe := grpc.NewServer()
		messaging.RegisterPathMessengerServer(peerServe, newPeerServer())
		peerServe.Serve(listen)
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

	conn, err := grpc.Dial(fmt.Sprintf("%v:%d", *registrationIP, *registrationPort), opts...)
	if err != nil {
		log.Fatalf("failed to connect to server %v:%d", *registrationIP, *registrationPort)
	}
	defer conn.Close()

	peerChan := make(chan string)
	startPeerService(peerChan)
	myIP := <-peerChan
	if strings.HasPrefix(myIP, "error") {
		log.Fatal(myIP)
	}

	regClient := messaging.NewOverlayRegistrationClient(conn)
	registerToOverlay(regClient, myIP)

	time.Sleep(20 * time.Second)
}
