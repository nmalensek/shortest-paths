package registration

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/golang/mock/gomock"
	"github.com/nmalensek/shortest-paths/messaging"
)

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockdialer(ctrl)
	opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
	s := New(opts)
	s.dial = m.DialFunc

	m.EXPECT().DialFunc(gomock.Any(), gomock.Any()).Return(&grpc.ClientConn{}, nil)

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
