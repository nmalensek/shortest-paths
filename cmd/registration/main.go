package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/nmalensek/shortest-paths/addressing"
	"github.com/nmalensek/shortest-paths/messaging"
	"github.com/nmalensek/shortest-paths/node/registration"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "", "the absolute path to the registration node config file in JSON format")
)

func main() {
	conf := setConfig()

	localIP := addressing.GetIP().String()

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%d", localIP, conf.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("Listening on %v:%d\n", localIP, conf.Port)

	grpcServer := grpc.NewServer()

	messaging.RegisterOverlayRegistrationServer(grpcServer,
		registration.New([]grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}, conf))

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func setConfig() registration.Config {
	flag.Parse()

	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("could not open config file: %v", err)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("could not read config file: %v", err)
	}

	var conf registration.Config
	json.Unmarshal(fileBytes, &conf)
	if err != nil {
		log.Fatalf("could not unmarshal config file: %v", err)
	}

	return conf
}
