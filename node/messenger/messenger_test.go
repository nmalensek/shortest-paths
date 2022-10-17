package messenger

import (
	"context"
	"fmt"
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
						MessageType: RECEIVED,
						Payload:     int32(payloadOne),
					}
				}
				wg.Done()
			}()

			go func() {
				for i := 0; i < sendTwo; i++ {
					s.statsChan <- recStats{
						MessageType: RECEIVED,
						Payload:     int32(payloadTwo),
					}
				}
				for j := 0; j < relayCount; j++ {
					s.statsChan <- recStats{
						MessageType: RELAYED,
					}
				}
				wg.Done()
			}()

			go s.trackReceivedData()

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
		name                      string
		numWorkers                int
		numSenders                int
		payloadAmount             int32
		numSinkMessagesPerSender  int
		numRelayMessagesPerSender int
		wantPayload               int64
		wantReceived              int64
		wantRelayed               int64
		otherNodes                map[string][]string
	}{
		{
			name:                      "correctly process sink messages",
			numSenders:                5,
			payloadAmount:             1000,
			numSinkMessagesPerSender:  100,
			numRelayMessagesPerSender: 0,
		},
		{
			name:                      "handle relay and sink messages correctly when path is known",
			numSenders:                10,
			payloadAmount:             1000,
			numSinkMessagesPerSender:  100,
			numRelayMessagesPerSender: 100,
			otherNodes: map[string][]string{
				"127.0.0.1:9999": {
					"127.0.0.1:8888",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOtherNode := messaging.NewMockPathMessengerClient(gomock.NewController(t))
			s := &MessengerServer{
				serverAddress: "127.0.0.1:8000",
				shutdownChan:  make(chan struct{}),
				statsChan:     make(chan recStats),
				nodePathDict:  tt.otherNodes,
				nodeConns: map[string]messaging.PathMessengerClient{
					"127.0.0.1:8888": mockOtherNode,
				},
			}

			go s.trackReceivedData()

			tt.wantReceived = int64(tt.numSenders) * int64(tt.numSinkMessagesPerSender)
			tt.wantPayload = tt.wantReceived * int64(tt.payloadAmount)
			tt.wantRelayed = int64(tt.numSenders) * int64(tt.numRelayMessagesPerSender)

			testStart := time.Now()

			wg := &sync.WaitGroup{}

			// message the running MessengerServer with messages where it's the destination.
			for i := 0; i < tt.numSenders; i++ {
				wg.Add(1)
				go func() {
					for i := 0; i < tt.numSinkMessagesPerSender; i++ {
						s.AcceptMessage(context.Background(), &messaging.PathMessage{
							Payload: tt.payloadAmount,
							Destination: &messaging.Node{
								Id: s.serverAddress,
							},
							Path: []*messaging.Node{},
						})
					}

					wg.Done()
				}()
			}

			for i := 0; i < tt.numSenders; i++ {
				wg.Add(1)
				go func() {
					for i := 0; i < tt.numRelayMessagesPerSender; i++ {
						mockOtherNode.EXPECT().AcceptMessage(gomock.Any(), &messaging.PathMessage{
							Payload: tt.payloadAmount,
							Destination: &messaging.Node{
								Id: "127.0.0.1:9999",
							},
							Path: []*messaging.Node{{Id: s.serverAddress}},
						}).Return(&messaging.PathResponse{}, nil)

						s.AcceptMessage(context.Background(), &messaging.PathMessage{
							Payload: tt.payloadAmount,
							Destination: &messaging.Node{
								Id: "127.0.0.1:9999",
							},
							Path: []*messaging.Node{},
						})
					}

					wg.Done()
				}()
			}

			// when running, nodes will finish their task and inform the registration node.
			// can't do that here so wait until expected messages are processed or it's been too long.
			wg.Add(1)
			go func() {
				for s.messagesReceived+s.messagesRelayed < tt.wantReceived+tt.wantRelayed {
					// shut down after a max of 3 seconds
					if time.Since(testStart).Seconds() >= 3 {
						close(s.shutdownChan)
						fmt.Printf("forced shut down, received: %v\tpayload: %v\t relayed:%v", s.messagesReceived, s.payloadReceived, s.messagesRelayed)
						return
					}
					time.Sleep(time.Millisecond * 10)
				}
				wg.Done()
			}()

			wg.Wait()

			m, _ := s.GetMessagingData(context.Background(), &messaging.MessagingDataRequest{})

			if tt.wantReceived != m.MessagesReceived {
				t.Fatalf("messages received differs, want %v got %v", tt.wantReceived, m.MessagesReceived)
			}

			if tt.wantPayload != m.PayloadReceived {
				t.Fatalf("payload received differs, want %v got %v", tt.wantPayload, m.PayloadReceived)
			}

			if tt.wantRelayed != m.MessagesRelayed {
				t.Fatalf("messages relayed differs, want %v got %v", tt.wantRelayed, m.MessagesRelayed)
			}
		})
	}
}

func TestMessengerServer_processMessagesUnknownPath(t *testing.T) {
	tests := []struct {
		name                      string
		numWorkers                int
		numSenders                int
		payloadAmount             int32
		numRelayMessagesPerSender int
		wantPayload               int64
		wantReceived              int64
		wantRelayed               int64
		otherNodes                map[string][]string
	}{
		{
			name:                      "handle discarding relay messages when path is unknown",
			numWorkers:                10,
			numSenders:                10,
			payloadAmount:             1000,
			numRelayMessagesPerSender: 100,
			wantPayload:               0,
			wantReceived:              0,
			wantRelayed:               0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MessengerServer{
				serverAddress: "127.0.0.1:8000",
				shutdownChan:  make(chan struct{}),
				statsChan:     make(chan recStats),
				nodePathDict:  tt.otherNodes,
			}

			go s.trackReceivedData()

			testStart := time.Now()

			wg := &sync.WaitGroup{}

			for i := 0; i < tt.numSenders; i++ {
				wg.Add(1)
				go func() {
					for i := 0; i < tt.numRelayMessagesPerSender; i++ {
						s.AcceptMessage(context.Background(), &messaging.PathMessage{
							Payload: tt.payloadAmount,
							Destination: &messaging.Node{
								Id: "127.0.0.1:11111111",
							},
							Path: []*messaging.Node{},
						})
					}

					wg.Done()
				}()
			}

			// when running, nodes will finish their task and inform the registration node.
			// can't do that here so wait until expected messages are processed or it's been too long.
			wg.Add(1)
			go func() {
				for s.messagesReceived+s.messagesRelayed < tt.wantReceived+tt.wantRelayed {
					// shut down after a max of 3 seconds
					if time.Since(testStart).Seconds() >= 3 {
						close(s.shutdownChan)
						fmt.Printf("forced shut down, received: %v\tpayload: %v\t relayed:%v", s.messagesReceived, s.payloadReceived, s.messagesRelayed)
						return
					}
					time.Sleep(time.Millisecond * 10)
				}
				wg.Done()
			}()

			wg.Wait()

			m, _ := s.GetMessagingData(context.Background(), &messaging.MessagingDataRequest{})

			if tt.wantReceived != m.MessagesReceived {
				t.Fatalf("messages received differs, want %v got %v", tt.wantReceived, m.MessagesReceived)
			}

			if tt.wantPayload != m.PayloadReceived {
				t.Fatalf("payload received differs, want %v got %v", tt.wantPayload, m.PayloadReceived)
			}

			if tt.wantRelayed != m.MessagesRelayed {
				t.Fatalf("messages relayed differs, want %v got %v", tt.wantRelayed, m.MessagesRelayed)
			}
		})
	}
}

func TestMessengerServer_doTask(t *testing.T) {
	tests := []struct {
		name        string
		otherNodes  map[string][]string
		numMessages int64
	}{
		{
			name: "send messages to another node successfully",
			otherNodes: map[string][]string{
				"127.0.0.1:9999": {"127.0.0.1:8888"},
			},
			numMessages: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRecipient := messaging.NewMockPathMessengerClient(ctrl)
			mockRegistrationClient := messaging.NewMockOverlayRegistrationClient(ctrl)
			s := &MessengerServer{
				nodePathDict: tt.otherNodes,
				nodeConns: map[string]messaging.PathMessengerClient{
					"127.0.0.1:8888": mockRecipient,
				},
				messagesSentRequirement: tt.numMessages,
				batchMessages:           5,
				registratonConn:         mockRegistrationClient,
				startTaskChan:           make(chan struct{}),
			}
			mockRegistrationClient.EXPECT().NodeFinished(gomock.Any(), &messaging.NodeStatus{Id: s.serverAddress, Status: messaging.NodeStatus_COMPLETE})

			for i := 0; i < int(tt.numMessages); i++ {
				mockRecipient.EXPECT().AcceptMessage(gomock.Any(), gomock.Any()).Return(&messaging.PathResponse{}, nil)
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)

			go func() {
				s.doTask()
				wg.Done()
			}()

			s.startTaskChan <- struct{}{}

			wg.Wait()

			if s.messagesSent != tt.numMessages {
				t.Fatalf("sent unexpected number of messages, got %v want %v", s.messagesSent, tt.numMessages)
			}
		})
	}
}
