package overlay

import (
	"reflect"
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
		{Source: nodes[0], Destination: nodes[1]},
		{Source: nodes[0], Destination: nodes[4]},
		{Source: nodes[0], Destination: nodes[2]},
		{Source: nodes[1], Destination: nodes[2]},
		{Source: nodes[1], Destination: nodes[5]},
		{Source: nodes[1], Destination: nodes[3]},
		{Source: nodes[2], Destination: nodes[3]},
		{Source: nodes[2], Destination: nodes[4]},
		{Source: nodes[3], Destination: nodes[4]},
		{Source: nodes[3], Destination: nodes[5]},
		{Source: nodes[4], Destination: nodes[5]},
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
			if got := buildOverlay(tt.args.nodeList, tt.args.reqConns, tt.args.randomize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildOverlay() = %v, want %v", got, tt.want)
			}
		})
	}
}
