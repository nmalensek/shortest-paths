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

	sixNodes := []*messaging.Node{
		{Id: "0"},
		{Id: "1"},
		{Id: "2"},
		{Id: "3"},
		{Id: "4"},
		{Id: "5"},
	}

	tests := []struct {
		name string
		args args
		want []*messaging.Edge
	}{
		{
			name: "non-random 6 node overlay with 4 connections",
			args: args{
				nodeList:  sixNodes,
				reqConns:  4,
				randomize: false,
			},
			want: []*messaging.Edge{
				{Source: sixNodes[0], Destination: sixNodes[5]},
				{Source: sixNodes[0], Destination: sixNodes[4]},
				{Source: sixNodes[0], Destination: sixNodes[1]},
				{Source: sixNodes[0], Destination: sixNodes[2]},
				{Source: sixNodes[1], Destination: sixNodes[5]},
				{Source: sixNodes[1], Destination: sixNodes[2]},
				{Source: sixNodes[1], Destination: sixNodes[3]},
				{Source: sixNodes[2], Destination: sixNodes[3]},
				{Source: sixNodes[2], Destination: sixNodes[4]},
				{Source: sixNodes[3], Destination: sixNodes[4]},
				{Source: sixNodes[3], Destination: sixNodes[5]},
				{Source: sixNodes[4], Destination: sixNodes[5]},
			},
		},
		{
			name: "non-random 4 node overlay with 4 connections desired; end with 3 connections each",
			args: args{
				nodeList:  sixNodes[:4],
				reqConns:  4,
				randomize: false,
			},
			want: []*messaging.Edge{
				{Source: sixNodes[0], Destination: sixNodes[3]},
				{Source: sixNodes[0], Destination: sixNodes[2]},
				{Source: sixNodes[0], Destination: sixNodes[1]},
				{Source: sixNodes[1], Destination: sixNodes[2]},
				{Source: sixNodes[1], Destination: sixNodes[3]},
				{Source: sixNodes[2], Destination: sixNodes[3]},
			},
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
