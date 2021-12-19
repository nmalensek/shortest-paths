package overlay

import (
	"testing"

	"github.com/nmalensek/shortest-paths/messaging"
)

func Test_buildOverlay(t *testing.T) {
	type args struct {
		nodeList  []*messaging.Node
		reqConns  int
		randomize bool
	}

	nodes := []*messaging.Node{
		{Id: "0"},
		{Id: "1"},
		{Id: "2"},
		{Id: "3"},
		{Id: "4"},
		{Id: "5"},
	}

	edges := []*messaging.Edge{
		{Source: nodes[0], Destination: nodes[5]},
		{Source: nodes[5], Destination: nodes[0]},

		{Source: nodes[0], Destination: nodes[4]},
		{Source: nodes[4], Destination: nodes[0]},

		{Source: nodes[0], Destination: nodes[1]},
		{Source: nodes[1], Destination: nodes[0]},

		{Source: nodes[0], Destination: nodes[2]},
		{Source: nodes[2], Destination: nodes[0]},

		{Source: nodes[1], Destination: nodes[5]},
		{Source: nodes[5], Destination: nodes[1]},

		{Source: nodes[1], Destination: nodes[2]},
		{Source: nodes[2], Destination: nodes[1]},

		{Source: nodes[1], Destination: nodes[3]},
		{Source: nodes[3], Destination: nodes[1]},

		{Source: nodes[2], Destination: nodes[3]},
		{Source: nodes[3], Destination: nodes[2]},

		{Source: nodes[2], Destination: nodes[4]},
		{Source: nodes[4], Destination: nodes[2]},

		{Source: nodes[3], Destination: nodes[4]},
		{Source: nodes[4], Destination: nodes[3]},

		{Source: nodes[3], Destination: nodes[5]},
		{Source: nodes[5], Destination: nodes[3]},

		{Source: nodes[4], Destination: nodes[5]},
		{Source: nodes[5], Destination: nodes[4]},
	}

	tests := []struct {
		name string
		args args
		want []*messaging.Edge
	}{
		{
			name: "non-random 6 node overlay with 4 connections",
			args: args{
				nodeList:  nodes,
				reqConns:  4,
				randomize: false,
			},
			want: edges,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildOverlay(tt.args.nodeList, tt.args.reqConns, tt.args.randomize)
			if len(got) != len(tt.want) {
				t.Errorf("connection count mismatch, got %v want %v", len(got), len(tt.want))
			}
			for _, e := range got {
				if !containsEdge(e, tt.want) {
					t.Errorf("have connection that is not expected: %v", e)
				}
			}
			for _, e := range tt.want {
				if !containsEdge(e, got) {
					t.Errorf("missing connection %v", e)
				}
			}

		})
	}
}

func containsEdge(e *messaging.Edge, edgeList []*messaging.Edge) bool {
	for _, edge := range edgeList {
		if e.Source.Id == edge.Source.Id && e.Destination.Id == edge.Destination.Id {
			return true
		}
	}
	return false
}
