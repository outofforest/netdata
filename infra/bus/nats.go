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
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/logger"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/retry"
	"go.uber.org/zap"
)

// NewNATSConnection creates new NATS connection
func NewNATSConnection(config infra.Config) Connection {
	opts := nats.GetDefaultOptions()
	opts.Url = strings.Join(config.NATSAddresses, ",")
	opts.Name = "Netdata"
	opts.Timeout = 10 * time.Second
	opts.PingInterval = 10 * time.Second
	opts.MaxPingsOut = 3
	opts.NoEcho = true
	opts.Verbose = config.VerboseLogging
	return &natsConnection{
		config: config,
		opts:   opts,
		subs:   map[interface{}]bool{},
		ready:  make(chan struct{}),
	}
}

// natsConnection is NATS-specific implementation of Connection interface
type natsConnection struct {
	config infra.Config
	opts   nats.Options
	nc     *nats.Conn
	ready  chan struct{}

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

// Subscribe subscribes to topic and decodes received messages
func (conn *natsConnection) Subscribe(ctx context.Context, templatePtr Entity) (OnRecvCh, error) {
	if err := waitReady(ctx, conn.ready); err != nil {
		return nil, err
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	topic := topicForValue(templatePtr)
	if conn.subs[topic] {
		return nil, fmt.Errorf("subscription to topic %s already exists", topic)
	}
	conn.subs[topic] = true

	log := logger.Get(ctx).With(zap.String("topic", topic))
	log.Info("Subscribing to topic")

	recvCh := make(chan chan<- struct{})
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
		templatePtr.SetShardPreID(deriveShardPreID(templatePtr.ShardSeed()))
		if shardID := templatePtr.ShardID(conn.config.NumOfShards); shardID != conn.config.ShardID {
			log.Debug("Entity not for this shard received, ignoring", zap.Any("dstShardID", shardID), zap.Any("shardID", conn.config.ShardID))
			return
		}

		msgCh := make(chan struct{})
		recvCh <- msgCh
		_ = waitReady(ctx, msgCh)
	}); err != nil {
		return nil, fmt.Errorf("subscription failed: %w", err)
	}

	log.Info("Subscribed to topic")
	return recvCh, nil
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
