package netdata

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ridge/parallel"
	"github.com/wojciech-malota-wojcik/netdata/infra/wire"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
	"go.uber.org/zap"
)

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

// runLocalShard runs a local shard
func runLocalShard(i uint64, rx <-chan interface{}, tx chan<- interface{}) parallel.Task {
	return func(ctx context.Context) error {
		log := logger.Get(ctx).With(zap.Uint64("localShardIndex", i))
		log.Info("Local shard started")

		users := userList{}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg, ok := <-rx:
				if !ok {
					return nil
				}
				log := log.With(zap.Any("msg", msg))

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

					if len(active.ActiveAlarms) == 0 {
						continue
					}

					sort.Slice(active.ActiveAlarms, func(i int, j int) bool {
						return active.ActiveAlarms[i].LatestChangedAt.Before(active.ActiveAlarms[j].LatestChangedAt)
					})

					tx <- active

					for _, alarm := range sent {
						alarm.ToSend = false
					}

					log.Info("Alarms sent", zap.Any("alarms", active))

				default:
					log.Warn(fmt.Sprintf("Message of unknown type %T received", msg))
				}
			}
		}
	}
}
