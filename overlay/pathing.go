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
		address: sourceAddr,
	}
	unvisited := PriorityQueue{node}
	heap.Init(&unvisited)

	visited := make(map[string]struct{})

	for {
		if len(unvisited) == 0 {
			return nil, fmt.Errorf("failed to calculate shortest path from %v to %v; unvisited is empty", sourceAddr, destAddr)
		}
		node = heap.Pop(&unvisited).(*graphEdge)
		node.path = append(node.path, node.address)

		if node.address == destAddr {
			// exclude first in path (start node)
			return node.path[1:], nil
		}

		visited[node.address] = struct{}{}
		for _, n := range connections[node.address] {
			// handle edges being bidirectional by using the end that's not the current node.
			target := n.Destination.Id
			if n.Destination.Id == node.address {
				target = n.Source.Id
			}

			_, inVisited := visited[target]
			item, inUnvisited := unvisited.Contains(target)
			if !inVisited && !inUnvisited {
				heap.Push(&unvisited, &graphEdge{
					address: target,
					weight:  int(n.Weight),
					path:    node.path,
				})
			} else if inUnvisited {
				current := item.(*graphEdge)
				if current.weight > int(n.Weight) {
					unvisited.update(current, current.address, int(n.Weight), node.path)
				}
			}
		}
	}
}

// Adapted from https://pkg.go.dev/container/heap@go1.17.5
// A graphEdge is something we manage in a priority queue.
type graphEdge struct {
	address string   // The value of the item; arbitrary.
	weight  int      // The priority of the item in the queue.
	path    []string // The node addresses making up the shortest path.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds graphEdges.
type PriorityQueue []*graphEdge

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].weight < pq[j].weight
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*graphEdge)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of a graphEdge in the queue.
func (pq *PriorityQueue) update(item *graphEdge, address string, weight int, newPath []string) {
	item.address = address
	item.weight = weight
	item.path = newPath
	heap.Fix(pq, item.index)
}

func (pq *PriorityQueue) Contains(address string) (interface{}, bool) {
	for _, i := range *pq {
		if i.address == address {
			return i, true
		}
	}
	return nil, false
}
