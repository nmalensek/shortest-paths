package messenger

import (
	"context"
	"reflect"
	"sync"
	"testing"

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
			mockRegClient := TestMockOverlayRegistrationClient{
				WaitGroup: &sync.WaitGroup{},
			}
			s := &MessengerServer{
				serverAddress:   "mockMessenger",
				registratonConn: mockRegClient,
			}

			if !tt.wantErr {
				mockRegClient.WaitGroup.Add(1)
			}

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

// TestMockOverlayRegistrationClient mocks OverlayRegistrationClient because the messenger calls doTask() -> NodeFinished() in a goroutine;
// this type allows us to use a WaitGroup to wait until that goroutine is done so the test correctly passes/fails without any race conditions.
type TestMockOverlayRegistrationClient struct {
	WaitGroup *sync.WaitGroup
}

func (t TestMockOverlayRegistrationClient) RegisterNode(ctx context.Context, in *messaging.Node, opts ...grpc.CallOption) (*messaging.RegistrationResponse, error) {
	return &messaging.RegistrationResponse{}, nil
}

func (t TestMockOverlayRegistrationClient) DeregisterNode(ctx context.Context, in *messaging.Node, opts ...grpc.CallOption) (*messaging.DeregistrationResponse, error) {
	return &messaging.DeregistrationResponse{}, nil
}

func (t TestMockOverlayRegistrationClient) GetOverlay(ctx context.Context, in *messaging.EdgeRequest, opts ...grpc.CallOption) (messaging.OverlayRegistration_GetOverlayClient, error) {
	return nil, nil
}

func (t TestMockOverlayRegistrationClient) ProcessMetadata(ctx context.Context, in *messaging.MessagingMetadata, opts ...grpc.CallOption) (*messaging.MetadataConfirmation, error) {
	return &messaging.MetadataConfirmation{}, nil
}

func (t TestMockOverlayRegistrationClient) NodeReady(ctx context.Context, in *messaging.NodeStatus, opts ...grpc.CallOption) (*messaging.TaskReadyResponse, error) {
	return &messaging.TaskReadyResponse{}, nil
}

func (t TestMockOverlayRegistrationClient) NodeFinished(ctx context.Context, in *messaging.NodeStatus, opts ...grpc.CallOption) (*messaging.TaskCompleteResponse, error) {
	t.WaitGroup.Done()
	return &messaging.TaskCompleteResponse{}, nil
}
