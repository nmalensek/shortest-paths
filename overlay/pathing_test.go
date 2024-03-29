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
			makeEdge("node1", "node2", 3),
			makeEdge("node2", "node3", 10),
			makeEdge("node2", "node4", 6),
		},
		"node3": {
			makeEdge("node2", "node3", 10),
			makeEdge("node3", "node4", 14),
			makeEdge("node1", "node3", 5),
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
		{
			name: "fail on empty overlay",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "na",
				connections: make(map[string][]*messaging.Edge),
			},
			want:    nil,
			wantErr: true,
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

func TestGetShortestPathsTridentOverlay(t *testing.T) {
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
			name: "successfully find shortest path from node 1 to node 9 in 10 node overlay",
			args: args{
				sourceAddr: "node1",
				destAddr:   "node9",
				connections: map[string][]*messaging.Edge{
					"node1": {
						makeEdge("node1", "node2", 40),
						makeEdge("node1", "node3", 40),
						makeEdge("node1", "node4", 43),
					},
					"node2": {
						makeEdge("node2", "node1", 40),
						makeEdge("node2", "node5", 10),
					},
					"node3": {
						makeEdge("node3", "node1", 40),
						makeEdge("node3", "node6", 9),
					},
					"node4": {
						makeEdge("node4", "node1", 43),
						makeEdge("node4", "node7", 22),
					},
					"node5": {
						makeEdge("node5", "node2", 10),
						makeEdge("node5", "node8", 1),
					},
					"node6": {
						makeEdge("node6", "node3", 9),
						makeEdge("node6", "node9", 3),
					},
					"node7": {
						makeEdge("node7", "node4", 22),
						makeEdge("node7", "node10", 17),
					},
					"node8": {
						makeEdge("node8", "node5", 1),
					},
					"node9": {
						makeEdge("node9", "node6", 3),
					},
					"node10": {
						makeEdge("node10", "node7", 17),
					},
				},
			},
			want:    []string{"node3", "node6", "node9"},
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

// successfully find shortest paths when each node has a direct connection that's a shortest path.
func TestGetShortestPathDirectConnection(t *testing.T) {
	edges := map[string][]*messaging.Edge{
		"node1": {
			makeEdge("node1", "node2", 40),
			makeEdge("node1", "node3", 6),
			makeEdge("node1", "node4", 10),
		},
		"node2": {
			makeEdge("node2", "node1", 40),
			makeEdge("node2", "node5", 10),
		},
		"node3": {
			makeEdge("node3", "node1", 6),
			makeEdge("node3", "node6", 9),
		},
		"node4": {
			makeEdge("node4", "node1", 10),
			makeEdge("node4", "node7", 22),
		},
		"node5": {
			makeEdge("node5", "node2", 10),
			makeEdge("node5", "node8", 1),
		},
		"node6": {
			makeEdge("node6", "node3", 9),
			makeEdge("node6", "node9", 3),
		},
	}

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
			name: "1 to 9",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "node9",
				connections: edges,
			},
			want:    []string{"node3", "node6", "node9"},
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

// Test against an overlay that was generated by overlay.go
func TestGetShortestGeneratedOverlay(t *testing.T) {
	edges := map[string][]*messaging.Edge{
		"node1": {
			makeEdge("node1", "node2", 10),
			makeEdge("node1", "node3", 22),
			makeEdge("node1", "node5", 14),
			makeEdge("node1", "node6", 49),
		},
		"node2": {
			makeEdge("node2", "node1", 10),
			makeEdge("node2", "node3", 16),
			makeEdge("node2", "node4", 46),
			makeEdge("node2", "node5", 19),
		},
		"node3": {
			makeEdge("node3", "node1", 22),
			makeEdge("node3", "node2", 16),
			makeEdge("node3", "node4", 37),
			makeEdge("node3", "node6", 30),
		},
		"node4": {
			makeEdge("node4", "node2", 46),
			makeEdge("node4", "node3", 37),
			makeEdge("node4", "node5", 4),
			makeEdge("node4", "node6", 36),
		},
		"node5": {
			makeEdge("node5", "node1", 14),
			makeEdge("node5", "node2", 19),
			makeEdge("node5", "node4", 4),
			makeEdge("node5", "node6", 46),
		},
		"node6": {
			makeEdge("node6", "node1", 49),
			makeEdge("node6", "node3", 30),
			makeEdge("node6", "node4", 36),
			makeEdge("node6", "node5", 46),
		},
	}

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
			name: "1 to 3",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "node3",
				connections: edges,
			},
			want:    []string{"node3"},
			wantErr: false,
		},
		{
			name: "1 to 6",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "node6",
				connections: edges,
			},
			want:    []string{"node6"},
			wantErr: false,
		},
		{
			name: "1 to 4",
			args: args{
				sourceAddr:  "node1",
				destAddr:    "node4",
				connections: edges,
			},
			want:    []string{"node5", "node4"},
			wantErr: false,
		},
		{
			name: "2 to 6",
			args: args{
				sourceAddr:  "node2",
				destAddr:    "node6",
				connections: edges,
			},
			want:    []string{"node3", "node6"},
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
