package main

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc/peer"

	"github.com/nmalensek/shortest-paths/messaging"
)

func TestRegister(t *testing.T) {
	s := registrationServer{registeredNodes: make(map[string]*messaging.Node)}
	addr := "127.0.0.1:9999"

	req := &messaging.Node{Id: addr}
	x, _ := net.ResolveIPAddr("tcp", "127.0.0.1:9999")

	p := peer.NewContext(context.Background(), &peer.Peer{Addr: x})

	_, err := s.RegisterNode(p, req)
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
		return
	}
}
