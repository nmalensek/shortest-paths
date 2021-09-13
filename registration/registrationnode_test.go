package registration

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nmalensek/shortest-paths/config"
	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)

	m := NewMockdialer(ctrl)
	opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
	s := New(opts, config.RegistrationServer{})
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

func TestRegistrationServer_DeregisterNode(t *testing.T) {
	type args struct {
		ctx context.Context
		n   *messaging.Node
	}
	tests := []struct {
		name        string
		args        args
		nodes       map[string]*messaging.Node
		overlaySent bool
		want        *messaging.DeregistrationResponse
		wantErr     bool
	}{
		{
			name: "successful deregister",
			args: args{
				ctx: context.Background(),
				n:   &messaging.Node{Id: "test:123"},
			},
			nodes: map[string]*messaging.Node{
				"test:123": {
					Id: "test:123",
				},
			},
			want:    &messaging.DeregistrationResponse{},
			wantErr: false,
		},
		{
			name: "missing deregister",
			args: args{
				ctx: context.Background(),
				n:   &messaging.Node{Id: "test:456"},
			},
			nodes: map[string]*messaging.Node{
				"test:123": {
					Id: "test:123",
				},
			},
			want:    &messaging.DeregistrationResponse{},
			wantErr: true,
		},
		{
			name: "overlay sent",
			args: args{
				ctx: context.Background(),
				n:   &messaging.Node{Id: "test:456"},
			},
			nodes: map[string]*messaging.Node{
				"test:123": {
					Id: "test:123",
				},
			},
			overlaySent: true,
			want:        &messaging.DeregistrationResponse{},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
			s := New(opts, config.RegistrationServer{})
			s.registeredNodes = tt.nodes
			s.overlaySent = tt.overlaySent

			got, err := s.DeregisterNode(tt.args.ctx, tt.args.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegistrationServer.DeregisterNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RegistrationServer.DeregisterNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
