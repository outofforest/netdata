package bus

import (
	"context"
)

// OnRecvCh is the channel where new channel is sent whenever message is decoded from topic
// received channel has to be closed just after decoded message is processed
type OnRecvCh <-chan chan<- struct{}

// Connection is an interface of event broker client
type Connection interface {
	// Run is a task which maintains and closes connection whenever context is canceled
	Run(ctx context.Context) error

	// Subscribe subscribes to the type-specific topic and receives messages from there
	Subscribe(ctx context.Context, templatePtr interface{}) (OnRecvCh, error)

	// Publish sends message to type-specific topic
	Publish(ctx context.Context, msg interface{}) error
}
