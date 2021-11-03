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
	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/infra/sharding"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
	"github.com/wojciech-malota-wojcik/netdata/lib/retry"
	"go.uber.org/zap"
)

// NewNATSConnection creates new NATS connection
func NewNATSConnection(config infra.Config, shardIDGen sharding.IDGenerator) Connection {
	opts := nats.GetDefaultOptions()
	opts.Url = strings.Join(config.NATSAddresses, ",")
	opts.Name = "Netdata"
	opts.Timeout = 10 * time.Second
	opts.PingInterval = 10 * time.Second
	opts.MaxPingsOut = 3
	opts.NoEcho = true
	opts.Verbose = config.VerboseLogging
	return &natsConnection{
		config:     config,
		shardIDGen: shardIDGen,
		opts:       opts,
		publishCh:  make(chan interface{}),
		ready:      make(chan struct{}),
	}
}

// natsConnection is NATS-specific implementation of Connection interface
type natsConnection struct {
	config     infra.Config
	shardIDGen sharding.IDGenerator
	opts       nats.Options
	publishCh  chan interface{}
	nc         *nats.Conn
	ready      chan struct{}
}

// Run is a task which closes connection whenever context is canceled
func (conn *natsConnection) Run(ctx context.Context) error {
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

	log.Info("Starting incoming loop")
	defer log.Info("Terminating NATS connection")

	for msg := range conn.publishCh {
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

// Subscribe returns task subscribing to the type-specific topic, receiving messages from there and distributing them between receiving channels
func (conn *natsConnection) Subscribe(ctx context.Context, templatePtr Entity, recvChs []chan<- interface{}) parallel.Task {
	return func(ctx context.Context) error {
		if err := waitReady(ctx, conn.ready); err != nil {
			return err
		}

		templateValue := reflect.ValueOf(templatePtr).Elem()
		topic := topicForValue(templatePtr)

		return parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
			log := logger.Get(ctx).With(zap.String("topic", topic))
			log.Info("Subscribing to topic")

			msgCh := make(chan *nats.Msg)
			sub, err := conn.nc.ChanSubscribe(topic, msgCh)
			if err != nil {
				return fmt.Errorf("subscription failed: %w", err)
			}

			spawn("handler", parallel.Fail, func(ctx context.Context) error {
				for m := range msgCh {
					log.Debug("Message received", zap.ByteString("msg", m.Data))
					if err := json.Unmarshal(m.Data, templatePtr); err != nil {
						log.Error("Decoding message failed", zap.Error(err))
						continue
					}
					if err := templatePtr.Validate(); err != nil {
						log.Error("Received entity is in invalid state", zap.Error(err))
						continue
					}
					shardIDs := conn.shardIDGen.Generate(templatePtr.ShardSeed(), conn.config.NumOfShards, uint64(len(recvChs)))
					if shardID := shardIDs[0]; shardID != conn.config.ShardID {
						log.Debug("Entity not for this shard received, ignoring", zap.Any("dstShardID", shardID), zap.Any("shardID", conn.config.ShardID))
						continue
					}

					localShardID := shardIDs[1]
					recvChs[localShardID] <- templateValue.Interface()
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

// PublishCh returns channel used to publish messages
func (conn *natsConnection) PublishCh() chan<- interface{} {
	return conn.publishCh
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
