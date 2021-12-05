package overlay

import (
	"math/rand"
	"time"

	"github.com/nmalensek/shortest-paths/messaging"
)

type graph struct {
}

func getShortestPaths() {

}

var weightGenerator = rand.New(rand.NewSource(time.Now().Unix()))

func randomWeight(rd *rand.Rand) int {
	return rd.Intn(51)
}

func buildOverlay(nodeList []*messaging.Node, reqConns int, randomize bool) []*messaging.Edge {
	connections := make(map[string][]*messaging.Edge, len(nodeList))
	// source nodeId <-> dest nodeId; used to track existing connections per node
	connMap := make(map[string]map[string]struct{}, len(nodeList))

	if randomize {
		rand.Shuffle(len(nodeList), func(i, j int) {
			nodeList[i], nodeList[j] = nodeList[j], nodeList[i]
		})
	}

	// connect each node to reqConns other nodes
	for _, n := range nodeList {
		targetNodeIndex := len(nodeList) - 1
		for len(connections[n.Id]) < reqConns {
			targetNode := nodeList[targetNodeIndex%len(nodeList)]

			// because edges are bi-directional, update and check both sides per connection.
			_, currentToOther := connMap[n.Id][targetNode.Id]
			_, otherToCurrent := connMap[targetNode.Id][n.Id]
			if n != targetNode && !currentToOther && !otherToCurrent {
				edge := &messaging.Edge{
					Source:      n,
					Destination: targetNode,
					Weight:      int32(randomWeight(weightGenerator)),
				}
				connections[n.Id] = append(connections[n.Id], edge)
				connMap[n.Id] = map[string]struct{}{
					targetNode.Id: {},
				}
				connMap[targetNode.Id] = map[string]struct{}{
					n.Id: {},
				}
			}

			targetNodeIndex++
		}
	}

	return []*messaging.Edge{}
}
