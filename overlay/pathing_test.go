package overlay

import (
	"reflect"
	"testing"

	"github.com/nmalensek/shortest-paths/messaging"
)

func makeEdge(source, dest string, weight int) *messaging.Edge {
	return &messaging.Edge{
		Source:      &messaging.Node{Id: source},
		Destination: &messaging.Node{Id: dest},
		Weight:      int32(weight),
	}
}

func makeOverlay() map[string][]*messaging.Edge {
	return map[string][]*messaging.Edge{
		"node1": {
			makeEdge("node1", "node2", 3),
			makeEdge("node1", "node6", 2),
			makeEdge("node1", "node3", 5),
		},
		"node2": {
			makeEdge("node2", "node1", 3),
			makeEdge("node2", "node3", 10),
			makeEdge("node2", "node4", 6),
		},
		"node3": {
			makeEdge("node3", "node2", 10),
			makeEdge("node3", "node4", 14),
			makeEdge("node3", "node1", 5),
		},
		"node4": {
			makeEdge("node4", "node3", 14),
			makeEdge("node4", "node2", 6),
			makeEdge("node4", "node5", 7),
		},
		"node5": {
			makeEdge("node5", "node4", 7),
			makeEdge("node5", "node6", 1),
		},
		"node6": {
			makeEdge("node6", "node5", 1),
			makeEdge("node6", "node1", 2),
		},
	}
}

func TestGetShortestPath(t *testing.T) {
	type args struct {
		sourceAddr  string
		destAddr    string
		connections map[string][]*messaging.Edge
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "successfully find shortest path from node1 to node6 in 6 node overlay",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "node6",
				connections: makeOverlay(),
			},
			want:    []string{"node6"},
			wantErr: false,
		},
		{
			name: "successfully find shortest path from node2 to node5 in 6 node overlay",
			args: args{
				sourceAddr:  "node2",
				destAddr:    "node5",
				connections: makeOverlay(),
			},
			want:    []string{"node1", "node6", "node5"},
			wantErr: false,
		},
		{
			name: "successfully find shortest path from node1 to node4 in 6 node overlay",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "node4",
				connections: makeOverlay(),
			},
			want:    []string{"node2", "node4"},
			wantErr: false,
		},
		{
			name: "successfully find shortest path from node3 to node5 in 6 node overlay",
			args: args{
				sourceAddr:  "node3",
				destAddr:    "node5",
				connections: makeOverlay(),
			},
			want:    []string{"node1", "node6", "node5"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetShortestPath(tt.args.sourceAddr, tt.args.destAddr, tt.args.connections)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetShortestPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetShortestPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
