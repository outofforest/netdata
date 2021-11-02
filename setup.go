package digest

import (
	"github.com/wojciech-malota-wojcik/ioc"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/logger"
)

// IoC configures IoC container
func IoC(c *ioc.Container) {
	c.Singleton(NewConfigFactory)
	c.Transient(func(configF *ConfigFactory) infra.Config {
		return configF.Config()
	})
}

// NewConfigFactory creates new ConfigFactory
func NewConfigFactory() *ConfigFactory {
	return &ConfigFactory{}
}

// ConfigFactory collects config from CLI and produces real config
type ConfigFactory struct {
	// VerboseLogging turns on verbose logging
	VerboseLogging bool
}

// Config produces final config
func (cf *ConfigFactory) Config() infra.Config {
	config := infra.Config{
		VerboseLogging: cf.VerboseLogging,
	}

	if !config.VerboseLogging {
		logger.VerboseOff()
	}

	return config
}
