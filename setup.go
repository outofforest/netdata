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
	pflag.BoolVarP(&cf.VerboseLogging, "verbose", "v", false, "Turns on verbose logging")
	pflag.Parse()
	return cf
}

// ConfigFactory collects config from CLI and produces real config
type ConfigFactory struct {
	// NATSAddresses contains addresses of NATS cluster
	NATSAddresses []string

	// VerboseLogging turns on verbose logging
	VerboseLogging bool
}

// Config produces final config
func (cf *ConfigFactory) Config() infra.Config {
	config := infra.Config{
		NATSAddresses:  cf.NATSAddresses,
		VerboseLogging: cf.VerboseLogging,
	}

	if !config.VerboseLogging {
		logger.VerboseOff()
	}

	return config
}
