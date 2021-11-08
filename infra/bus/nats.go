package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/ridge/must"
	"github.com/ridge/parallel"
	"github.com/wojciech-malota-wojcik/logger"
	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/lib/retry"
	"go.uber.org/zap"
)

// NewNATSConnection creates new NATS connection
func NewNATSConnection(config infra.Config, dispatcherF DispatcherFactory) Connection {
	opts := nats.GetDefaultOptions()
	opts.Url = strings.Join(config.NATSAddresses, ",")
	opts.Name = "Netdata"
	opts.Timeout = 10 * time.Second
	opts.PingInterval = 10 * time.Second
	opts.MaxPingsOut = 3
	opts.NoEcho = true
	opts.Verbose = config.VerboseLogging
	return &natsConnection{
		config:      config,
		dispatcherF: dispatcherF,
		opts:        opts,
		ready:       make(chan struct{}),
	}
}

// natsConnection is NATS-specific implementation of Connection interface
type natsConnection struct {
	config      infra.Config
	dispatcherF DispatcherFactory
	opts        nats.Options
	nc          *nats.Conn
	ready       chan struct{}
}

// Run is a task which maintains and closes connection
func (conn *natsConnection) Run(publishCh <-chan interface{}) parallel.Task {
	return func(ctx context.Context) error {
		log := logger.Get(ctx).With(zap.String("servers", conn.opts.Url))
		log.Info("Connecting to NATS")

		_ = retry.Do(ctx, time.Second, func() error {
			var err error
			conn.nc, err = conn.opts.Connect()
			if err != nil {
				return retry.Retryable(fmt.Errorf("can't connect to NATS: %w", err))
			}
			return nil
		})
		defer conn.nc.Close()

		log.Info("Connected to NATS")
		close(conn.ready)

		defer log.Info("Terminating NATS connection")

		log.Info("Starting outgoing loop")

		for msg := range publishCh {
			log.Debug("Sending message", zap.Any("msg", msg))

			// Publish method of NATS doesn't send message over the network, it only buffers it.
			// If it fails it means there is a serious problem on the server (no more memory etc.)
			// So I decided it's better to panic in this case rather than hide the real issue in retry loop
			if err := conn.nc.Publish(topicForValue(msg), must.Bytes(json.Marshal(msg))); err != nil {
				panic(err)
			}
		}

		return ctx.Err()
	}
}

// Subscribe returns task subscribing to the type-specific topic, receiving messages from there and distributing them between receiving channels
func (conn *natsConnection) Subscribe(ctx context.Context, templatePtr Entity, recvChs []chan<- interface{}) parallel.Task {
	return func(ctx context.Context) error {
		if err := waitReady(ctx, conn.ready); err != nil {
			return err
		}

		return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
			topic := topicForValue(templatePtr)

			log := logger.Get(ctx).With(zap.String("topic", topic))
			log.Info("Subscribing to topic")

			dispatcher := conn.dispatcherF.Create(templatePtr, recvChs, log)

			msgCh := make(chan *nats.Msg)
			sub, err := conn.nc.ChanSubscribe(topic, msgCh)
			if err != nil {
				return fmt.Errorf("subscription failed: %w", err)
			}

			spawn("handler", parallel.Fail, func(ctx context.Context) error {
				for m := range msgCh {
					dispatcher.Dispatch(ctx, m.Data)
				}
				return ctx.Err()
			})
			spawn("closer", parallel.Fail, func(ctx context.Context) error {
				defer close(msgCh)

				<-ctx.Done()
				if err := sub.Drain(); err != nil {
					return err
				}
				return ctx.Err()
			})

			log.Info("Subscribed to topic")
			return nil
		})
	}
}

func topicForValue(val interface{}) string {
	t := reflect.TypeOf(val)
	if t.Kind() != reflect.Ptr {
		panic(fmt.Errorf("type %T is not a pointer", val))
	}
	tName := t.String()
	return tName[strings.LastIndex(tName, ".")+1:]
}

func waitReady(ctx context.Context, ch <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}
