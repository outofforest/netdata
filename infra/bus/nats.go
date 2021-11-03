package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/ridge/must"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/sharding"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/logger"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/retry"
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
		subs:       map[interface{}]bool{},
		ready:      make(chan struct{}),
	}
}

// natsConnection is NATS-specific implementation of Connection interface
type natsConnection struct {
	config     infra.Config
	shardIDGen sharding.IDGenerator
	opts       nats.Options
	nc         *nats.Conn
	ready      chan struct{}

	mu   sync.Mutex
	subs map[interface{}]bool
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

	<-ctx.Done()

	log.Info("Terminating NATS connection")

	return ctx.Err()
}

// Subscribe subscribes to the type-specific topic, receives messages from there and distributes them between receiving channels
func (conn *natsConnection) Subscribe(ctx context.Context, templatePtr Entity, recvChs []RecvCh) error {
	if err := waitReady(ctx, conn.ready); err != nil {
		return err
	}

	templateValue := reflect.ValueOf(templatePtr).Elem()

	conn.mu.Lock()
	defer conn.mu.Unlock()

	topic := topicForValue(templatePtr)
	if conn.subs[topic] {
		return fmt.Errorf("subscription to topic %s already exists", topic)
	}
	conn.subs[topic] = true

	log := logger.Get(ctx).With(zap.String("topic", topic))
	log.Info("Subscribing to topic")

	if _, err := conn.nc.Subscribe(topic, func(m *nats.Msg) {
		log.Debug("Message received", zap.ByteString("msg", m.Data))
		if err := json.Unmarshal(m.Data, templatePtr); err != nil {
			log.Error("Decoding message failed", zap.Error(err))
			return
		}
		if err := templatePtr.Validate(); err != nil {
			log.Error("Received entity is in invalid state", zap.Error(err))
			return
		}
		shardIDs := conn.shardIDGen.Generate(templatePtr.ShardSeed(), conn.config.NumOfShards, uint64(len(recvChs)))
		if shardID := shardIDs[0]; shardID != conn.config.ShardID {
			log.Debug("Entity not for this shard received, ignoring", zap.Any("dstShardID", shardID), zap.Any("shardID", conn.config.ShardID))
			return
		}

		localShardID := shardIDs[1]
		recvChs[localShardID] <- templateValue.Interface()
	}); err != nil {
		return fmt.Errorf("subscription failed: %w", err)
	}

	log.Info("Subscribed to topic")
	return nil
}

// Publish sends message to the topic
func (conn *natsConnection) Publish(ctx context.Context, msg interface{}) error {
	if err := waitReady(ctx, conn.ready); err != nil {
		return err
	}
	return conn.nc.Publish(topicForValue(msg), must.Bytes(json.Marshal(msg)))
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
