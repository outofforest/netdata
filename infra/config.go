package infra

// Config stores configuration
type Config struct {
	// ShardID is the shard ID of the node
	ShardID ShardID

	// NumOfShards is the total number of running shards
	NumOfShards uint64

	// NATSAddresses contains addresses of NATS cluster
	NATSAddresses []string

	// VerboseLogging turns on verbose logging
	VerboseLogging bool
}
