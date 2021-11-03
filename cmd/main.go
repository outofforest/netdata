package main

import (
	"context"

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
			spawn("incomingUpdates", parallel.Fail, func(ctx context.Context) error {
				update := &wire.AlarmStatusChanged{}
				updatesRecvCh, err := conn.Subscribe(ctx, update)
				if err != nil {
					return err
				}

				log := logger.Get(ctx)
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case doneCh := <-updatesRecvCh:
						func() {
							defer close(doneCh)

							log := log.With(zap.Any("update", update))
							log.Info("Update received")
						}()
					}
				}
			})
			spawn("incomingRequests", parallel.Fail, func(ctx context.Context) error {
				send := &wire.SendAlarmDigest{}
				sendRecvCh, err := conn.Subscribe(ctx, send)
				if err != nil {
					return err
				}

				log := logger.Get(ctx)
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case doneCh := <-sendRecvCh:
						func() {
							defer close(doneCh)

							log := log.With(zap.Any("send", send))
							log.Info("Send request received")
						}()
					}
				}
			})
			return nil
		})
	})
}
