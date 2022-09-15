package overlay

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/nmalensek/shortest-paths/messaging"
)

var weightGenerator = rand.New(rand.NewSource(time.Now().Unix()))

func randomWeight(rd *rand.Rand) int {
	return rd.Intn(51)
}

// BuildOverlay builds an overlay using the given list of nodes and randomizes connections if desired.
// reqConns will be attempted to be fulfilled but is best effort if an incompatible reqConns is given relative
// to the size of nodeList.
func BuildOverlay(nodeList []*messaging.Node, reqConns int, randomize bool) []*messaging.Edge {
	// storing as a map so it's easier to pass the edge slice to the makeConnection func
	connections := make(map[string][]*messaging.Edge, len(nodeList))
	// source nodeId <-> dest nodeId; used to track existing connections per node
	connMap := make(map[string]map[string]struct{}, len(nodeList))

	if randomize {
		rand.Shuffle(len(nodeList), func(i, j int) {
			nodeList[i], nodeList[j] = nodeList[j], nodeList[i]
		})
	}

	// connect to nodes on either side of the current node, roughly half behind and half in front
	startPoint := (reqConns / 2)

	for i, n := range nodeList {
		targetIndex := ((i + len(nodeList)) - startPoint) % len(nodeList)
		checkedSelf := false

		for len(connMap[n.Id]) < reqConns {
			// nodes can't connect to themselves
			if targetIndex == i {

				// if we get to this node a second time, we can't hit reqConns
				if checkedSelf {
					fmt.Printf("node %v cannot find new connections, aborting with the following connections: %v\n",
						n.Id, connections[n.Id])
					break
				}

				checkedSelf = true
			} else {
				targetNode := nodeList[targetIndex]

				// because edges are bi-directional, update and check both sides per connection.
				_, currentToOther := connMap[n.Id][targetNode.Id]
				_, otherToCurrent := connMap[targetNode.Id][n.Id]
				if !currentToOther && !otherToCurrent {
					makeConnection(connections, connMap, n, targetNode)
				}
			}

			targetIndex = (targetIndex + 1) % len(nodeList)

		}
	}

	edges := make([]*messaging.Edge, 0, len(nodeList))
	for _, v := range connections {
		edges = append(edges, v...)
	}

	return edges
}

func makeConnection(edgeMap map[string][]*messaging.Edge, connMap map[string]map[string]struct{},
	sourceNode *messaging.Node, targetNode *messaging.Node) {
	edge := &messaging.Edge{
		Source:      sourceNode,
		Destination: targetNode,
		Weight:      int32(randomWeight(weightGenerator)),
		// Weight:      0,
	}

	edgeMap[sourceNode.Id] = append(edgeMap[sourceNode.Id], edge)

	if connMap[sourceNode.Id] == nil {
		connMap[sourceNode.Id] = make(map[string]struct{})
	}
	connMap[sourceNode.Id][targetNode.Id] = struct{}{}

	if connMap[targetNode.Id] == nil {
		connMap[targetNode.Id] = make(map[string]struct{})
	}
	connMap[targetNode.Id][sourceNode.Id] = struct{}{}
}
