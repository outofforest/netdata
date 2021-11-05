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
	"github.com/wojciech-malota-wojcik/netdata/lib/libctx"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
)

const localShardBufferSize = 100

// IoCBuilder configures IoC container
func IoCBuilder(c *ioc.Container) {
	c.Singleton(infra.NewConfigFromCLI)
	c.Transient(sharding.NewXORModuloIDGenerator)
	c.Transient(bus.NewDispatcherFactory)
	c.Singleton(bus.NewNATSConnection)
}

// App is the main function running application logic
func App(ctx context.Context, config infra.Config, conn bus.Connection) error {
	if !config.VerboseLogging {
		logger.VerboseOff()
	}

	return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
		tx := make(chan interface{})
		rxes := make([]chan interface{}, 0, config.NumOfLocalShards)
		for i := uint64(0); i < config.NumOfLocalShards; i++ {
			rx := make(chan interface{}, localShardBufferSize)
			rxes = append(rxes, rx)
		}

		spawn("bus", parallel.Fail, conn.Run(tx))
		spawn("localShards", parallel.Fail, func(ctx context.Context) error {
			defer close(tx)

			ctx, cancel := context.WithCancel(libctx.Reopen(ctx))
			defer cancel()

			return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
				for i, rx := range rxes {
					spawn(fmt.Sprintf("%d", i), parallel.Continue, runLocalShard(uint64(i), rx, tx))
				}
				return nil
			})
		})
		spawn("subscriptions", parallel.Fail, func(ctx context.Context) error {
			// Converting []chan interface{} to []chan<- interface{}
			txes := make([]chan<- interface{}, 0, len(rxes))
			for _, tx := range rxes {
				txes = append(txes, tx)
			}

			defer func() {
				for _, tx := range txes {
					close(tx)
				}
			}()

			return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
				spawn("subscription-rx", parallel.Fail, conn.Subscribe(ctx, &wire.AlarmStatusChanged{}, txes))
				spawn("subscription-tx", parallel.Fail, conn.Subscribe(ctx, &wire.SendAlarmDigest{}, txes))
				return nil
			})
		})
		return nil
	})
}
