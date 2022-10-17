# shortest-paths
Sending messages along the shortest path using gRPC and protocol buffers. This project is a re-implementation of a Java homework project using Dijkstra's algorithm to find the shortest path between nodes. The primary goal of this project was to try out gRPC and protocol buffers.

## Process Flow
All nodes register with a single Registration Node, which generates a random overlay once the desired number of nodes have registered. Overlay paths are given random weights and the the final overlay is pushed out to the registered nodes. The nodes determine which neighbors they need to connect to based on the overlay, and once done, inform the Registration Node they are ready. Once all nodes are ready, the Registration Node directs all nodes to start sending messages by selecting another random node in the overlay as the destination and sending to the node that's the first hop in the shortest path. Number of messages is specified in the Registration Node's configuration file. Message payloads are simply a random signed 64-bit integer.

Once all nodes have completed their task, the Registration Node requests statistics from each node and prints out the result (number of messages sent, received, and relayed; payload totals; and elapsed time).

## Shortest Path Algorithm
Shortest paths are calculated using the Uniform Cost Search, a simplified version of Dijkstra's algorithm that only calculates shortest paths from one node at a time:  https://en.wikipedia.org/wiki/Dijkstra%27s_algorithm#Practical_optimizations_and_infinite_graphs

The UCS algorithm was selected so each node could concurrently calculate its own shortests paths to each other node in the overlay without doing throwaway work.

## Design Notes
Originally, nodes had a worker pool implemented following the example here: https://pkg.go.dev/golang.org/x/sync/semaphore. The worker pool was used to process received messages; however, this meant sending was very cheap (the sender's gRPC call returned immediately) and quickly exhausted the worker pool. In overlays with six nodes, this quickly resulted in messages timing out because there was such a backlog of messages to get through, especially when the overlay happened to be set up so that all relay messages passed through the same 1-2 nodes. Removing the woker pool and requiring the sender to wait until a receiver had processed the message before sending another solved the problem, presumably because senders synchronized around receivers' statistics channels.

Relays are still sent using new goroutines so that a sender will not block for an excessively long amount of time given a long path to a destination node; senders simply care that the first hop was transmitted successfully in this design. 
