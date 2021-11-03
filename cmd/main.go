package main

import (
	"context"
	"fmt"

	"github.com/wojciech-malota-wojcik/netdata-digest/infra/sharding"

	"github.com/ridge/parallel"
	"github.com/wojciech-malota-wojcik/ioc"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/bus"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/wire"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/logger"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/run"
	"go.uber.org/zap"
)

const localShardBufferSize = 100

// iocBuilder configures IoC container
func iocBuilder(c *ioc.Container) {
	c.Singleton(infra.NewConfigFromCLI)
	c.Transient(sharding.NewXORModuloIDGenerator)
	c.Transient(bus.NewNATSConnection)
}

func main() {
	run.Service("digest", iocBuilder, func(ctx context.Context, config infra.Config, conn bus.Connection) error {
		if !config.VerboseLogging {
			logger.VerboseOff()
		}

		return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
			spawn("bus", parallel.Fail, conn.Run)

			localShardChs := make([]bus.RecvCh, 0, config.NumOfLocalShards)
			for i := uint64(0); i < config.NumOfLocalShards; i++ {
				localShardCh := make(chan interface{}, localShardBufferSize)
				localShardChs = append(localShardChs, localShardCh)

				spawn(fmt.Sprintf("localShard-%d", i), parallel.Fail, localShard(i, localShardCh))
			}

			if err := conn.Subscribe(ctx, &wire.AlarmStatusChanged{}, localShardChs); err != nil {
				return err
			}
			return conn.Subscribe(ctx, &wire.SendAlarmDigest{}, localShardChs)
		})
	})
}

func localShard(i uint64, ch <-chan interface{}) parallel.Task {
	return func(ctx context.Context) error {
		log := logger.Get(ctx).With(zap.Uint64("localShardIndex", i))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg := <-ch:
				log := log.With(zap.Any("message", msg))
				switch msg.(type) {
				case wire.AlarmStatusChanged:
					log.Info("Update received")
				case wire.SendAlarmDigest:
					log.Info("Request received")
				default:
					panic(fmt.Errorf("message of unknown type %T received", msg))
				}
			}
		}
	}
}
