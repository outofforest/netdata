package digest

import (
	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
	"github.com/wojciech-malota-wojcik/ioc"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/bus"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/logger"
)

// IoC configures IoC container
func IoC(c *ioc.Container) {
	c.Singleton(NewConfigFactoryFromCLI)
	c.Transient(func(configF *ConfigFactory) infra.Config {
		return configF.Config()
	})
	c.Transient(bus.NewNATSConnection)
}

// NewConfigFactoryFromCLI creates new ConfigFactory
func NewConfigFactoryFromCLI() *ConfigFactory {
	cf := &ConfigFactory{}
	pflag.StringSliceVar(&cf.NATSAddresses, "nats-addr", []string{nats.DefaultURL}, "Addresses of NATS cluster")
	pflag.Uint64Var(&cf.ShardID, "shard-id", 0, "Shard ID of node")
	pflag.Uint64Var(&cf.NumOfShards, "shards", 1, "Total number of running shards")
	pflag.BoolVarP(&cf.VerboseLogging, "verbose", "v", false, "Turns on verbose logging")
	pflag.Parse()

	if cf.ShardID >= cf.NumOfShards {
		panic("shard ID has to be less than number of shards")
	}

	return cf
}

// ConfigFactory collects config from CLI and produces real config
type ConfigFactory struct {
	// ShardID is the shard ID of the node
	ShardID uint64

	// NumOfShards is the total number of running shards
	NumOfShards uint64

	// NATSAddresses contains addresses of NATS cluster
	NATSAddresses []string

	// VerboseLogging turns on verbose logging
	VerboseLogging bool
}

// Config produces final config
func (cf *ConfigFactory) Config() infra.Config {
	config := infra.Config{
		ShardID:        infra.ShardID(cf.ShardID),
		NumOfShards:    cf.NumOfShards,
		NATSAddresses:  cf.NATSAddresses,
		VerboseLogging: cf.VerboseLogging,
	}

	if !config.VerboseLogging {
		logger.VerboseOff()
	}

	return config
}
