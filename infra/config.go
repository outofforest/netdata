package infra

import (
	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
)

// ShardID is the ID of the shard
type ShardID uint64

// NewConfigFromCLI creates new config based on CLI flags
func NewConfigFromCLI() Config {
	cfg := Config{}
	var shardID uint64
	pflag.StringSliceVar(&cfg.NATSAddresses, "nats-addr", []string{nats.DefaultURL}, "Addresses of NATS cluster")
	pflag.Uint64Var(&shardID, "shard-id", 0, "Shard ID of node")
	pflag.Uint64Var(&cfg.NumOfShards, "shards", 1, "Total number of running shards")
	pflag.BoolVarP(&cfg.VerboseLogging, "verbose", "v", false, "Turns on verbose logging")
	pflag.Parse()

	cfg.ShardID = ShardID(shardID)

	if shardID >= cfg.NumOfShards {
		panic("shard ID has to be less than number of shards")
	}

	return cfg
}

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
