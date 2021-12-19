package config

type RegistrationServer struct {
	// The server port
	Port int

	// The number of 5-message batches every other node in the overlay needs to send
	Rounds int

	// The number of nodes every other node must be connected to
	Connections int

	// The total number of nodes that need to be present in the overlay before starting the task
	Peers int
}
