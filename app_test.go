package netdata

import (
	"context"
	"sync"
	"testing"

	"github.com/ridge/parallel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/infra/bus"
	"github.com/wojciech-malota-wojcik/netdata/infra/wire"
	"github.com/wojciech-malota-wojcik/netdata/lib/libctx"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
)

type busConn struct {
	cancel          context.CancelFunc
	updatesRecvChs  []chan<- interface{}
	requestsRecvChs []chan<- interface{}

	ready1 chan struct{}
	ready2 chan struct{}

	mu      sync.Mutex
	digests map[wire.UserID][]wire.AlarmDigest
}

// Run is a task which maintains and closes connection
func (c *busConn) Run(publishCh <-chan interface{}) parallel.Task {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(libctx.Reopen(ctx))
		defer cancel()

		return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
			spawn("digests", parallel.Fail, func(ctx context.Context) error {
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case msg, ok := <-publishCh:
						if !ok {
							return nil
						}
						ad, ok := msg.(*wire.AlarmDigest)
						if !ok {
							panic("wrong type")
						}

						c.mu.Lock()
						c.digests[ad.UserID] = append(c.digests[ad.UserID], *ad)
						c.mu.Unlock()
					}
				}
			})
			spawn("inputs", parallel.Continue, func(ctx context.Context) error {
				defer c.cancel()

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-c.ready1:
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-c.ready2:
				}

				if err := deliver(ctx, c.updatesRecvChs[0], c.requestsRecvChs[0],
					change(user1, alarm1, wire.StatusWarning, time2),
					change(user1, alarm2, wire.StatusCritical, time1),
					send(user1)); err != nil {
					return err
				}

				if err := deliver(ctx, c.updatesRecvChs[1], c.requestsRecvChs[1],
					change(user2, alarm1, wire.StatusWarning, time1),
					change(user2, alarm1, wire.StatusCleared, time2),
					change(user2, alarm1, wire.StatusCritical, time4),
					change(user2, alarm1, wire.StatusWarning, time3),
					change(user2, alarm2, wire.StatusCritical, time2),
					send(user2),
					change(user2, alarm2, wire.StatusWarning, time3),
					send(user2)); err != nil {
					return err
				}

				return deliver(ctx, c.updatesRecvChs[2], c.requestsRecvChs[2],
					change(user3, alarm1, wire.StatusWarning, time1),
					change(user3, alarm2, wire.StatusCritical, time2),
					change(user3, alarm2, wire.StatusCleared, time3),
					send(user3))
			})
			return ctx.Err()
		})
	}
}

// Subscribe returns task subscribing to the type-specific topic, receiving messages from there and distributing them between receiving channels
func (c *busConn) Subscribe(ctx context.Context, templatePtr bus.Entity, recvChs []chan<- interface{}) parallel.Task {
	switch templatePtr.(type) {
	case *wire.AlarmStatusChanged:
		c.updatesRecvChs = recvChs
		close(c.ready1)
	case *wire.SendAlarmDigest:
		c.requestsRecvChs = recvChs
		close(c.ready2)
	default:
		panic("invalid subscription")
	}
	return func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
}

func deliver(ctx context.Context, updates chan<- interface{}, requests chan<- interface{}, msgs ...interface{}) error {
	for _, msg := range msgs {
		switch m := msg.(type) {
		case wire.AlarmStatusChanged:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case updates <- m:
			}
		case wire.SendAlarmDigest:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case requests <- m:
			}
		default:
			panic("invalid message")
		}
	}
	return nil
}

func TestApp(t *testing.T) {
	ctx, cancel := context.WithCancel(logger.WithLogger(context.Background(), logger.New()))
	t.Cleanup(cancel)

	config := infra.Config{
		NumOfLocalShards: 3,
	}
	conn := &busConn{
		cancel:  cancel,
		ready1:  make(chan struct{}),
		ready2:  make(chan struct{}),
		digests: map[wire.UserID][]wire.AlarmDigest{},
	}

	require.ErrorIs(t, context.Canceled, App(ctx, config, conn))
	assert.Equal(t, map[wire.UserID][]wire.AlarmDigest{
		user1: {
			{
				UserID: user1,
				ActiveAlarms: []wire.Alarm{
					{
						AlarmID:         alarm2,
						Status:          wire.StatusCritical,
						LatestChangedAt: time1,
					},
					{
						AlarmID:         alarm1,
						Status:          wire.StatusWarning,
						LatestChangedAt: time2,
					},
				},
			},
		},
		user2: {
			{
				UserID: user2,
				ActiveAlarms: []wire.Alarm{
					{
						AlarmID:         alarm2,
						Status:          wire.StatusCritical,
						LatestChangedAt: time2,
					},
					{
						AlarmID:         alarm1,
						Status:          wire.StatusCritical,
						LatestChangedAt: time4,
					},
				},
			},
			{
				UserID: user2,
				ActiveAlarms: []wire.Alarm{
					{
						AlarmID:         alarm2,
						Status:          wire.StatusWarning,
						LatestChangedAt: time3,
					},
				},
			},
		},
		user3: {
			{
				UserID: user3,
				ActiveAlarms: []wire.Alarm{
					{
						AlarmID:         alarm1,
						Status:          wire.StatusWarning,
						LatestChangedAt: time1,
					},
				},
			},
		},
	}, conn.digests)
}
