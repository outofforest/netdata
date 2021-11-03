package main

import (
	"context"
	"fmt"
	"sort"
	"time"

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

				spawn(fmt.Sprintf("localShard-%d", i), parallel.Fail, localShard(i, localShardCh, conn))
			}

			if err := conn.Subscribe(ctx, &wire.AlarmStatusChanged{}, localShardChs); err != nil {
				return err
			}
			return conn.Subscribe(ctx, &wire.SendAlarmDigest{}, localShardChs)
		})
	})
}

type userList map[wire.UserID]alarmList

type alarmList map[wire.AlarmID]*alarmStatus

type alarmStatus struct {
	// Status is the last reported status
	Status wire.Status

	// LatestChangedAt is the time when status was updated
	LatestChangedAt time.Time

	// ToSend is true if current state should be sent next time
	ToSend bool
}

func localShard(i uint64, ch <-chan interface{}, conn bus.Connection) parallel.Task {
	return func(ctx context.Context) error {
		log := logger.Get(ctx).With(zap.Uint64("localShardIndex", i))

		users := userList{}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg := <-ch:
				log := log.With(zap.Any("message", msg))
				switch m := msg.(type) {
				case wire.AlarmStatusChanged:
					alarms := users[m.UserID]
					if alarms == nil {
						alarms = alarmList{}
						users[m.UserID] = alarms
					}
					alarm := alarms[m.AlarmID]
					if alarm == nil {
						alarm = &alarmStatus{}
						alarms[m.AlarmID] = alarm
					}

					if alarm.LatestChangedAt.After(m.ChangedAt) {
						log.Info("Update ignored because newer one exists")
						continue
					}
					alarm.LatestChangedAt = m.ChangedAt

					switch {
					case alarm.Status != m.Status:
						alarm.Status = m.Status
						if alarm.Status == wire.StatusCleared {
							log.Info(fmt.Sprintf("Status is %s, alarm won't be sent", alarm.Status))
							alarm.ToSend = false
						} else {
							log.Info("Alarm triggered")
							alarm.ToSend = true
						}
					case alarm.ToSend:
						log.Info("Status hasn't changed, alarm was triggered earlier")
					default:
						log.Info("Status hasn't changed, alarm won't be sent")
					}
				case wire.SendAlarmDigest:
					log := log.With(zap.Any("userID", m.UserID))

					alarms := users[m.UserID]
					if alarms == nil {
						log.Info("No alarms for user, nothing to send")
					}

					active := &wire.AlarmDigest{
						UserID: m.UserID,
					}
					var sent []*alarmStatus
					for alarmID, alarm := range alarms {
						if alarm.ToSend {
							active.ActiveAlarms = append(active.ActiveAlarms, wire.Alarm{
								AlarmID:         alarmID,
								Status:          alarm.Status,
								LatestChangedAt: alarm.LatestChangedAt,
							})
							sent = append(sent, alarm)
						}
					}

					sort.Slice(active.ActiveAlarms, func(i int, j int) bool {
						return active.ActiveAlarms[i].LatestChangedAt.Before(active.ActiveAlarms[j].LatestChangedAt)
					})

					// Publish method of NATS doesn't send message over the network, it only buffers it.
					// If it fails it means there is a serious problem on the server (no more memory etc.)
					// So I decided it's better to panic in this case rather than hide the real issue in retry loop
					if err := conn.Publish(ctx, active); err != nil {
						panic(err)
					}

					for _, alarm := range sent {
						alarm.ToSend = false
					}

				default:
					panic(fmt.Errorf("message of unknown type %T received", msg))
				}
			}
		}
	}
}
