package overlay

import (
	"container/heap"
	"fmt"

	"github.com/nmalensek/shortest-paths/messaging"
)

func GetAllShortestPaths(source string, dests map[string]struct{}, connections map[string][]*messaging.Edge) (map[string][]string, error) {
	allPaths := make(map[string][]string)

	for destNode := range dests {
		res, err := GetShortestPath(source, destNode, connections)
		if err != nil {
			return nil, fmt.Errorf("could not calculate shortest paths for node %v; failed calculating path to %v: %v",
				source, destNode, err)
		}

		allPaths[destNode] = res
	}

	return allPaths, nil
}

// GetShortestPath calculates the shortest path from source to destination given a slice of edges.
func GetShortestPath(sourceAddr string, destAddr string, connections map[string][]*messaging.Edge) ([]string, error) {
	node := &graphEdge{
		Address: sourceAddr,
	}
	unvisited := PriorityQueue{node}
	heap.Init(&unvisited)

	visited := make(map[string]struct{})

	for {
		if len(unvisited) == 0 {
			return nil, fmt.Errorf("failed to calculate shortest path from %v to %v; unvisited is empty", sourceAddr, destAddr)
		}
		node = heap.Pop(&unvisited).(*graphEdge)
		node.Path = append(node.Path, node.Address)

		if node.Address == destAddr {
			// exclude first in path (start node)
			return node.Path[1:], nil
		}

		visited[node.Address] = struct{}{}
		for _, n := range connections[node.Address] {
			// handle edges being bidirectional by using the end that's not the current node.
			target := n.Destination.Id
			if n.Destination.Id == node.Address {
				target = n.Source.Id
			}

			_, inVisited := visited[target]
			item, inUnvisited := unvisited.Contains(target)
			if !inVisited && !inUnvisited {
				heap.Push(&unvisited, &graphEdge{
					Address: target,
					Weight:  int(n.Weight),
					Path:    node.Path,
				})
			} else if inUnvisited {
				knownEdge := item.(*graphEdge)
				// add previous weight?
				if knownEdge.Weight > int(n.Weight) {
					unvisited.update(knownEdge, knownEdge.Address, int(n.Weight), node.Path)
				}
			}
		}
	}
}

// Adapted from https://pkg.go.dev/container/heap@go1.17.5
// A graphEdge is something we manage in a priority queue.
type graphEdge struct {
	Address string   // The value of the item; arbitrary.
	Weight  int      // The priority of the item in the queue.
	Path    []string // The node addresses making up the shortest path.
	// The Index is needed by update and is maintained by the heap.Interface methods.
	Index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds graphEdges.
type PriorityQueue []*graphEdge

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Weight < pq[j].Weight
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*graphEdge)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of a graphEdge in the queue.
func (pq *PriorityQueue) update(item *graphEdge, address string, weight int, newPath []string) {
	item.Address = address
	item.Weight = weight
	item.Path = newPath
	heap.Fix(pq, item.Index)
}

func (pq *PriorityQueue) Contains(address string) (interface{}, bool) {
	for _, i := range *pq {
		if i.Address == address {
			return i, true
		}
	}
	return nil, false
}
