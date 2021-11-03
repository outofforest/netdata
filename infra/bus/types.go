package bus

import (
	"context"

	"github.com/ridge/parallel"
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
	// Run is a task which maintains and closes connection whenever context is canceled
	Run(ctx context.Context) error

	// Subscribe returns task subscribing to the type-specific topic, receiving messages from there and distributing them between receiving channels
	Subscribe(ctx context.Context, templatePtr Entity, recvChs []chan<- interface{}) parallel.Task

	// PublishCh returns channel used to publish messages
	PublishCh() chan<- interface{}
}
