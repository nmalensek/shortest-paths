package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/nmalensek/shortest-path/addressing"
	"github.com/nmalensek/shortest-path/messaging"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type registrationServer struct {
	messaging.UnimplementedOverlayRegistrationServer
	mu              sync.Mutex //prevent registration overwrites
	registeredNodes map[string]*messaging.Node
}

func makeServer() *registrationServer {
	return &registrationServer{registeredNodes: make(map[string]*messaging.Node)}
}

func main() {
	flag.Parse()

	localIP := addressing.GetIP().String()

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%d", localIP, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	messaging.RegisterOverlayRegistrationServer(grpcServer, makeServer())
	grpcServer.Serve(listener)
}
