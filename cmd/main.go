package main

import (
	"github.com/wojciech-malota-wojcik/netdata"
	"github.com/wojciech-malota-wojcik/netdata/infra/sharding"

	"github.com/wojciech-malota-wojcik/ioc"
	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/infra/bus"
	"github.com/wojciech-malota-wojcik/netdata/lib/run"
)

// iocBuilder configures IoC container
func iocBuilder(c *ioc.Container) {
	c.Singleton(infra.NewConfigFromCLI)
	c.Transient(sharding.NewXORModuloIDGenerator)
	c.Transient(bus.NewNATSConnection)
}

func main() {
	run.Service("digest", iocBuilder, netdata.App)
}
