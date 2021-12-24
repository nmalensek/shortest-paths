package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
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
	registrationIP   = flag.String("regIp", "127.0.0.1", "The IP address of the registration node")
	registrationPort = flag.Int("regPort", 10000, "The port the registration node is using")

	helpText = `this is the help text that will list the available commands eventually`
)

func startPeerService(c chan string) {
	addr := addressing.GetIP()
	go func() {
		listen, err := net.Listen("tcp", fmt.Sprintf("%v:0", addr.String()))
		if err != nil {
			c <- "error: failed to start the peer service"
		}
		c <- listen.Addr().String()

		peerServe := grpc.NewServer()
		messaging.RegisterPathMessengerServer(peerServe, messenger.New())
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
			regClient.DeregisterNode(context.Background(), &messaging.Node{Id: myIP})
			os.Exit(0)
		default:
			fmt.Printf("command %v not recognized, available options are:\n%v\n", command, helpText)
		}
	}
}
