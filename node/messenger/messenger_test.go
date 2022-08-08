package messenger

import (
	"context"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nmalensek/shortest-paths/messaging"
	"google.golang.org/grpc"
)

func TestMessengerServer_StartTask(t *testing.T) {
	type args struct {
		tr *messaging.TaskRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *messaging.TaskConfirmation
		wantErr bool
	}{
		{
			name: "valid requirements - no messages",
			args: args{
				tr: &messaging.TaskRequest{
					BatchesToSend:    0,
					MessagesPerBatch: 0,
				},
			},
			want: &messaging.TaskConfirmation{},
		},
		{
			name: "negative batches to send",
			args: args{
				tr: &messaging.TaskRequest{
					BatchesToSend:    -1,
					MessagesPerBatch: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "negative messages per batch",
			args: args{
				tr: &messaging.TaskRequest{
					BatchesToSend:    10,
					MessagesPerBatch: -5,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRegClient := messaging.NewMockOverlayRegistrationClient(gomock.NewController(t))
			s := &MessengerServer{
				serverAddress:   "mockMessenger",
				registratonConn: mockRegClient,
				startTaskChan:   make(chan struct{}),
			}

			// simulate receiving from the channel
			go func() {
				<-s.startTaskChan
			}()

			got, err := s.StartTask(context.Background(), tt.args.tr)
			if (err != nil) != tt.wantErr {
				t.Errorf("MessengerServer.StartTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MessengerServer.StartTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessengerServer_trackReceivedData(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "calculate stats correctly with concurrent modification",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MessengerServer{
				statsChan:    make(chan recStats),
				shutdownChan: make(chan struct{}),
			}

			sendOne := 10
			payloadOne := 100000

			sendTwo := 20
			payloadTwo := 200000

			relayCount := 5

			wg := sync.WaitGroup{}
			wg.Add(2)

			go func() {
				for i := 0; i < sendOne; i++ {
					s.statsChan <- recStats{
						messageReceived: true,
						payload:         int32(payloadOne),
					}
				}
				wg.Done()
			}()

			go func() {
				for i := 0; i < sendTwo; i++ {
					s.statsChan <- recStats{
						messageReceived: true,
						payload:         int32(payloadTwo),
					}
				}
				for j := 0; j < relayCount; j++ {
					s.statsChan <- recStats{
						messageRelayed: true,
					}
				}
				wg.Done()
			}()

			go s.trackReceivedData(s.statsChan, s.shutdownChan)

			wg.Wait()

			s.shutdownChan <- struct{}{}

			stats, err := s.GetMessagingData(context.Background(), &messaging.MessagingDataRequest{})
			if err != nil {
				t.Fatal("error getting stats")
			}

			if stats.MessagesReceived != (int64(sendOne) + int64(sendTwo)) {
				t.Fatalf("unexpected number of messages received, got %v want %v", stats.MessagesReceived, (sendOne + sendTwo))
			}

			if stats.PayloadReceived != int64((sendOne*payloadOne)+(sendTwo*payloadTwo)) {
				t.Fatalf("unexpected payload amount received, got %v want %v", stats.PayloadReceived, ((sendOne * payloadOne) + (sendTwo * payloadTwo)))
			}

			if stats.MessagesRelayed != int64(relayCount) {
				t.Fatalf("unexpected 'relayed' amount received, got %v want %v", stats.MessagesRelayed, relayCount)
			}
		})
	}
}

func TestMessengerServer_PushPaths(t *testing.T) {
	type args struct {
		stream messaging.PathMessenger_PushPathsServer
	}
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "process paths correctly",
		},
		{
			name:    "abort on context canceled",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MessengerServer{
				pathChan:     make(chan struct{}),
				overlayEdges: make([]*messaging.Edge, 0, 1),
			}
			tpss := newTestPathStreamServer()

			if tt.wantErr {
				tpss.Cancel()
			} else {
				go func() {
					<-s.pathChan
				}()
			}

			if err := s.PushPaths(tpss); (err != nil) != tt.wantErr {
				t.Errorf("MessengerServer.PushPaths() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testPathsStreamServer struct {
	grpc.ServerStream
	ctx        context.Context
	cancelFunc context.CancelFunc
	clientRecv chan *messaging.Edge
	index      int
	mockData   []*messaging.Edge
}

func newTestPathStreamServer() *testPathsStreamServer {
	cancelContext, cancel := context.WithCancel(context.Background())
	return &testPathsStreamServer{
		ctx:        cancelContext,
		cancelFunc: cancel,
		clientRecv: make(chan *messaging.Edge),
		mockData: []*messaging.Edge{
			{
				Source: &messaging.Node{
					Id: "abc123",
				},
				Destination: &messaging.Node{
					Id: "def456",
				},
				Weight: 1,
			},
			{
				Source: &messaging.Node{
					Id: "def456",
				},
				Destination: &messaging.Node{
					Id: "ghi789",
				},
				Weight: 1,
			},
			{
				Source: &messaging.Node{
					Id: "ghi789",
				},
				Destination: &messaging.Node{
					Id: "jkl012",
				},
				Weight: 1,
			},
			{
				Source: &messaging.Node{
					Id: "jkl012",
				},
				Destination: &messaging.Node{
					Id: "def456",
				},
				Weight: 1,
			},
		},
	}
}

func (s *testPathsStreamServer) Recv() (*messaging.Edge, error) {
	if s.index < len(s.mockData) {
		d := s.mockData[s.index]
		s.index++
		return d, nil
	}
	return nil, io.EOF
}

func (s *testPathsStreamServer) SendAndClose(*messaging.ConnectionResponse) error {
	return nil
}

func (s *testPathsStreamServer) Context() context.Context {
	return s.ctx
}

func (s *testPathsStreamServer) Cancel() {
	s.cancelFunc()
}

func TestMessengerServer_processMessages(t *testing.T) {
	tests := []struct {
		name          string
		numWorkers    int
		payloadAmount int32
		numMessages   int
		wantPayload   int64
		wantReceived  int64
		wantRelayed   int64
	}{
		{
			name:          "only process sink messages when path to relay node is unknown",
			numWorkers:    10,
			payloadAmount: 1000,
			numMessages:   20,
			wantPayload:   1000 * 10,
			wantReceived:  10,
			wantRelayed:   0,
		},
		// {
		// 	name:          "handle relay and sink messages correctly",
		// 	numWorkers:    10,
		// 	payloadAmount: 1000,
		// 	numMessages:   200,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MessengerServer{
				serverAddress: "127.0.0.1:8000",
				shutdownChan:  make(chan struct{}),
				statsChan:     make(chan recStats),
			}

			s.setWorkValues(tt.numWorkers)
			go s.trackReceivedData(s.statsChan, s.shutdownChan)

			testStart := time.Now()

			mockOtherNode := messaging.NewMockPathMessengerClient(gomock.NewController(t))

			go messageTargetNode("127.0.0.1:8000", "127.0.0.1:9999", mockOtherNode, s.workChan, tt.numMessages, tt.payloadAmount)

			go messageTargetNode("127.0.0.1:8000", "127.0.0.1:9999", mockOtherNode, s.workChan, tt.numMessages, tt.payloadAmount)

			go messageTargetNode("127.0.0.1:8000", "127.0.0.1:9999", mockOtherNode, s.workChan, tt.numMessages, tt.payloadAmount)

			go func() {
				for s.messagesReceived+s.messagesRelayed < tt.wantReceived+tt.wantRelayed {
					// force shutdown after a max of 5 seconds
					if time.Since(testStart).Seconds() >= 5 {
						close(s.shutdownChan)
						return
					}
					time.Sleep(time.Millisecond * 50)
				}
				close(s.shutdownChan)
			}()

			s.processMessages(s.workChan, s.statsChan, s.shutdownChan)
		})
	}
}

func messageTargetNode(targetSink string, targetRelayAddress string, targetRelayNode *messaging.MockPathMessengerClient,
	targetChan chan *messaging.PathMessage, numMessages int, payloadAmount int32) {

	for i := 0; i < numMessages; i++ {
		if i%2 == 0 {
			targetChan <- &messaging.PathMessage{
				Payload: payloadAmount,
				Destination: &messaging.Node{
					Id: targetSink,
				},
				Path: []*messaging.Node{},
			}
		} else {
			// targetRelayNode.EXPECT().AcceptMessage(gomock.Any(), &messaging.PathMessage{
			// 	Payload: payloadAmount,
			// 	Destination: &messaging.Node{
			// 		Id: targetRelayAddress,
			// 	},
			// 	Path: []*messaging.Node{{Id: targetSink}},
			// }).Return(&messaging.PathResponse{}, nil)

			targetChan <- &messaging.PathMessage{
				Payload: payloadAmount,
				Destination: &messaging.Node{
					Id: targetRelayAddress,
				},
				Path: []*messaging.Node{},
			}
		}
	}
}
