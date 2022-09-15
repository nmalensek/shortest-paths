package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/nmalensek/shortest-paths/addressing"
	"github.com/nmalensek/shortest-paths/node/messenger"

	"github.com/nmalensek/shortest-paths/messaging"

	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "", "the absolute path to the registration node config file in JSON format")
)

func main() {
	flag.Parse()

	conf := setConfig()

	opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}

	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", conf.RegistrationIP, conf.RegistrationPort), opts...)
	if err != nil {
		log.Fatalf("failed to connect to server %v:%v", conf.RegistrationIP, conf.RegistrationPort)
	}
	defer conn.Close()

	regClient := messaging.NewOverlayRegistrationClient(conn)

	// startPeerService starts a GRPC server on a random open port in a goroutine, capture its address
	// and use it to register to the overlay.
	peerChan := make(chan string)
	startPeerService(regClient, peerChan, conf.LogLevel)
	myIP := <-peerChan
	if strings.HasPrefix(myIP, "error") {
		log.Fatal(myIP)
	}
	fmt.Printf("Listening on %v\n", myIP)

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
			regClient.DeregisterNode(context.Background(), &messaging.Node{Id: myIP})
			os.Exit(0)
		default:
			fmt.Printf("command %v not recognized, available options are:\n%v\n", command, "exit")
		}
	}
}

func setConfig() messenger.Config {
	flag.Parse()

	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("could not open config file: %v", err)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("could not read config file: %v", err)
	}

	var conf messenger.Config
	json.Unmarshal(fileBytes, &conf)
	if err != nil {
		log.Fatalf("could not unmarshal config file: %v", err)
	}

	return conf
}

func startPeerService(regConn messaging.OverlayRegistrationClient, c chan string, logLevel string) {
	addr := addressing.GetIP()
	go func() {
		listen, err := net.Listen("tcp", fmt.Sprintf("%v:0", addr.String()))
		if err != nil {
			c <- "error: failed to start the peer service"
		}
		c <- listen.Addr().String()

		peerServe := grpc.NewServer()
		messaging.RegisterPathMessengerServer(peerServe, messenger.New(listen.Addr().String(), regConn, logLevel))
		serveErr := peerServe.Serve(listen)
		if serveErr != nil {
			log.Fatalf("server error: %v", serveErr)
		}
	}()
}

func registerToOverlay(c messaging.OverlayRegistrationClient, serveAddr string) error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*2))
	defer cancel()

	_, err := c.RegisterNode(ctx, &messaging.Node{Id: serveAddr})
	if err != nil {
		return err
	}
	return nil
}
