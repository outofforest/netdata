package bus

import (
	"context"

	"github.com/ridge/parallel"
	"go.uber.org/zap"
)

// Entity is implemented by structures which may be received from event bus
type Entity interface {
	// ShardSeed generates a seed used to compute shard ID
	ShardSeed() []byte

	// Validate validates if message contains valid data
	Validate() error
}

// Connection is an interface of event broker client
type Connection interface {
	// Run is a task which maintains and closes connection
	Run(publishCh <-chan interface{}) parallel.Task

	// Subscribe returns task subscribing to the type-specific topic, receiving messages from there and distributing them between receiving channels
	Subscribe(ctx context.Context, templatePtr Entity, recvChs []chan<- interface{}) parallel.Task
}

// Dispatcher decodes, validates and sends message to local shard
type Dispatcher interface {
	Dispatch(ctx context.Context, msg []byte)
}

// DispatcherFactory creates dispatchers
type DispatcherFactory interface {
	Create(templatePtr Entity, recvChs []chan<- interface{}, log *zap.Logger) Dispatcher
}
