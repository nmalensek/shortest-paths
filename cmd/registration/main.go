package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/nmalensek/shortest-paths/addressing"
	"github.com/nmalensek/shortest-paths/config"
	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/registration"
	"google.golang.org/grpc"
)

var (
	port        = flag.Int("port", 10000, "The server port")
	rounds      = flag.Int("rounds", 1000, "The number of 5-message batches every other node in the overlay needs to send")
	connections = flag.Int("connections", 4, "The number of nodes every other node must be connected to")
	peers       = flag.Int("peers", 10, "The total number of nodes that need to be present in the overlay before starting the task")
	grpcOpts    = []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
)

func setConfig() config.RegistrationServer {
	flag.Parse()

	return config.RegistrationServer{
		Port:        *port,
		Rounds:      *rounds,
		Connections: *connections,
		Peers:       *peers,
	}
}

func main() {
	localIP := addressing.GetIP().String()

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%d", localIP, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("Listening on %v:%d\n", localIP, *port)

	grpcServer := grpc.NewServer()
	conf := setConfig()

	messaging.RegisterOverlayRegistrationServer(grpcServer, registration.New(grpcOpts, conf))
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}
