package messenger

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nmalensek/shortest-paths/messaging"
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
			name: "calculate stats correctly",
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
				wg.Done()
			}()

			go s.trackReceivedData(s.statsChan, s.shutdownChan)

			wg.Wait()

			s.shutdownChan <- struct{}{}

			if s.messagesReceived != (int64(sendOne) + int64(sendTwo)) {
				t.Fatalf("unexpected number of messages received, got %v want %v", s.messagesReceived, (sendOne + sendTwo))
			}

			if s.payloadReceived != int64((sendOne*payloadOne)+(sendTwo*payloadTwo)) {
				t.Fatalf("unexpected payload amount received, got %v want %v", s.payloadReceived, ((sendOne * payloadOne) + (sendTwo * payloadTwo)))
			}
		})
	}
}
