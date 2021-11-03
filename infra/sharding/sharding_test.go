package sharding

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveShardIDSingleCase(t *testing.T) {
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

func TestDeriveShardIDDoubleCase(t *testing.T) {
	const numOfShards1 uint64 = 100
	const numOfShards2 uint64 = 10

	counts1 := make([]int, numOfShards1)
	counts2 := make([]int, numOfShards2)

	generator := NewXORModuloIDGenerator()
	for i := 0; i < 1000; i++ {
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		shardIDs := generator.Generate(id[:], numOfShards1, numOfShards2)
		assert.Less(t, shardIDs[0], numOfShards1)
		assert.Less(t, shardIDs[1], numOfShards2)
		counts1[shardIDs[0]]++
		counts2[shardIDs[1]]++
	}
	for _, c := range counts1 {
		assert.Greater(t, c, 0)
	}
	for _, c := range counts2 {
		assert.Greater(t, c, 0)
	}
}

func TestDeriveShardIDIsDeterministic(t *testing.T) {
	const numOfShards uint64 = 100

	id, err := uuid.NewRandom()
	require.NoError(t, err)

	generator1 := NewXORModuloIDGenerator()
	generator2 := NewXORModuloIDGenerator()

	shardIDs1 := generator1.Generate(id[:], numOfShards)
	shardIDs2 := generator2.Generate(id[:], numOfShards)

	assert.Equal(t, shardIDs1[0], shardIDs2[0])
}

func TestDeriveShardIDSimple(t *testing.T) {
	generator := NewXORModuloIDGenerator()
	assert.Equal(t, ID(0), generator.Generate([]byte{0x00}, 5)[0])
	assert.Equal(t, ID(1), generator.Generate([]byte{0x01}, 5)[0])
	assert.Equal(t, ID(2), generator.Generate([]byte{0x02}, 5)[0])
	assert.Equal(t, ID(3), generator.Generate([]byte{0x03}, 5)[0])
	assert.Equal(t, ID(4), generator.Generate([]byte{0x04}, 5)[0])
	assert.Equal(t, ID(0), generator.Generate([]byte{0x05}, 5)[0])
	assert.Equal(t, ID(1), generator.Generate([]byte{0x06}, 5)[0])
	assert.Equal(t, ID(2), generator.Generate([]byte{0x07}, 5)[0])
	assert.Equal(t, ID(3), generator.Generate([]byte{0x08}, 5)[0])
	assert.Equal(t, ID(4), generator.Generate([]byte{0x09}, 5)[0])
}
