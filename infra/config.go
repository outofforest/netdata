package infra

import (
	"runtime"

	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/sharding"
)

// NewConfigFromCLI creates new config based on CLI flags
func NewConfigFromCLI() Config {
	cfg := Config{}
	var shardID uint64
	pflag.StringSliceVar(&cfg.NATSAddresses, "nats-addr", []string{nats.DefaultURL}, "Addresses of NATS cluster")
	pflag.Uint64Var(&shardID, "shard-id", 0, "Shard ID of node")
	pflag.Uint64Var(&cfg.NumOfShards, "shards", 1, "Total number of shards managed by all nodes")
	pflag.Uint64Var(&cfg.NumOfLocalShards, "local-shards", uint64(runtime.NumCPU()), "Number of local shards")
	pflag.BoolVarP(&cfg.VerboseLogging, "verbose", "v", false, "Turns on verbose logging")
	pflag.Parse()

	cfg.ShardID = sharding.ID(shardID)

	if shardID >= cfg.NumOfShards {
		panic("shard ID has to be less than number of shards")
	}

	return cfg
}

// Config stores configuration
type Config struct {
	// ShardID is the shard ID of the node
	ShardID sharding.ID

	// NumOfShards is the total number of running shards
	NumOfShards uint64

	// NumOfLocalShards is the number of shards managed using local resources
	NumOfLocalShards uint64

	// NATSAddresses contains addresses of NATS cluster
	NATSAddresses []string

	// VerboseLogging turns on verbose logging
	VerboseLogging bool
}
