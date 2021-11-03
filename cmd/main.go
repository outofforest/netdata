package main

import (
	"context"

	"github.com/ridge/parallel"
	digest "github.com/wojciech-malota-wojcik/netdata-digest"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/bus"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/wire"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/logger"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/run"
	"go.uber.org/zap"
)

func main() {
	run.Service("digest", digest.IoC, func(ctx context.Context, config infra.Config, conn bus.Connection) error {
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
