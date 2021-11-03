package bus

import (
	"context"
)

// Entity is implemented by structures which may be received from event bus
type Entity interface {
	// ShardSeed generates a seed used to compute shard ID
	ShardSeed() []byte

	// Validate validates if message contains valid data
	Validate() error
}

// RecvCh is the channel where incomming messages are sent
type RecvCh chan<- interface{}

// Connection is an interface of event broker client
type Connection interface {
	// Run is a task which maintains and closes connection whenever context is canceled
	Run(ctx context.Context) error

	// Subscribe subscribes to the type-specific topic, receives messages from there and distributes them between receiving channels
	Subscribe(ctx context.Context, templatePtr Entity, recvChs []RecvCh) error

	// Publish sends message to type-specific topic
	Publish(ctx context.Context, msg interface{}) error
}
