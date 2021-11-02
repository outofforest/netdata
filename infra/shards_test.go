package infra

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wojciech-malota-wojcik/netdata-digest/infra/wire"
)

func TestDeriveShardID(t *testing.T) {
	counts := map[ShardID]int{}
	const numOfShards uint64 = 100
	for i := 0; i < 1000; i++ {
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		shardID := DeriveShardID(wire.UserID(id[:]), numOfShards)
		assert.Less(t, shardID, numOfShards)
		counts[shardID]++
	}
	for _, c := range counts {
		assert.NotEqual(t, 0, c)
	}
}
