package bus

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/infra/sharding"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
)

type deterministicShardIDGenerator struct {
	ids []sharding.ID
}

func (g *deterministicShardIDGenerator) Generate(seed []byte, counts ...uint64) []sharding.ID {
	return g.ids[:len(counts)]
}

type entity struct {
	err       error
	shardSeed []byte
}

func (e *entity) ShardSeed() []byte {
	return e.shardSeed
}

func (e entity) Validate() error {
	return e.err
}

func TestDispatcherFactory(t *testing.T) {
	// Setup

	config := infra.Config{
		ShardID:          3,
		NumOfShards:      5,
		NumOfLocalShards: 3,
	}

	shardIDGen := &deterministicShardIDGenerator{
		ids: []sharding.ID{3, 1},
	}

	chs := make([]chan interface{}, 0, config.NumOfLocalShards)
	recvChs := make([]chan<- interface{}, 0, config.NumOfLocalShards)

	for i := uint64(0); i < config.NumOfLocalShards; i++ {
		ch := make(chan interface{}, 1)
		chs = append(chs, ch)
		recvChs = append(recvChs, ch)
	}

	e := &entity{}

	df := NewDispatcherFactory(config, shardIDGen)
	disp := df.Create(e, recvChs, logger.New())

	// Action 1 - correct channel

	disp.Dispatch([]byte("{}"))
	require.Len(t, chs[1], 1)
	assert.Equal(t, *e, <-chs[1])

	// Action 2 - correct channel

	shardIDGen.ids = []sharding.ID{3, 2}

	disp.Dispatch([]byte("{}"))
	require.Len(t, chs[2], 1)
	assert.Equal(t, *e, <-chs[2])

	// Action 3 - invalid json

	disp.Dispatch([]byte("{"))
	assert.Len(t, chs[2], 0)

	// Action 4 - invalid entity

	e.err = errors.New("error")

	disp.Dispatch([]byte("{}"))
	assert.Len(t, chs[2], 0)

	// Action 5 - not my shard

	e.err = nil
	config.ShardID = 1
	df = NewDispatcherFactory(config, shardIDGen)
	disp = df.Create(e, recvChs, logger.New())

	disp.Dispatch([]byte("{}"))
	assert.Len(t, chs[2], 0)
}
