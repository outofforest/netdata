package netdata

import (
	"context"
	"fmt"

	"github.com/ridge/parallel"
	"github.com/wojciech-malota-wojcik/ioc"
	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/infra/bus"
	"github.com/wojciech-malota-wojcik/netdata/infra/sharding"
	"github.com/wojciech-malota-wojcik/netdata/infra/wire"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
)

const localShardBufferSize = 100

// IoCBuilder configures IoC container
func IoCBuilder(c *ioc.Container) {
	c.Singleton(infra.NewConfigFromCLI)
	c.Transient(sharding.NewXORModuloIDGenerator)
	c.Singleton(bus.NewNATSConnection)
}

// App is the main function running application logic
func App(ctx context.Context, config infra.Config, conn bus.Connection) error {
	if !config.VerboseLogging {
		logger.VerboseOff()
	}

	return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
		spawn("bus", parallel.Fail, conn.Run)

		tx := conn.PublishCh()
		rxs := make([]chan<- interface{}, 0, config.NumOfLocalShards)
		for i := uint64(0); i < config.NumOfLocalShards; i++ {
			rx := make(chan interface{}, localShardBufferSize)
			rxs = append(rxs, rx)

			spawn(fmt.Sprintf("localShard-%d", i), parallel.Fail, runLocalShard(i, rx, tx))
		}

		if err := conn.Subscribe(ctx, &wire.AlarmStatusChanged{}, rxs); err != nil {
			return err
		}
		return conn.Subscribe(ctx, &wire.SendAlarmDigest{}, rxs)
	})
}
