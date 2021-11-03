package sharding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveShardID(t *testing.T) {
	const numOfShards uint64 = 100
	counts := make([]int, numOfShards)
	generator := NewXORModuloIDGenerator()
	for i := 0; i < 1000; i++ {
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		shardIDs := generator.Generate(id[:], numOfShards)
		assert.Less(t, shardIDs[0], numOfShards)
		counts[shardIDs[0]]++
	}
	for _, c := range counts {
		assert.Greater(t, c, 0)
	}
}
