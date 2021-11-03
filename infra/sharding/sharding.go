package sharding

import "encoding/binary"

// ID is the ID of the shard
type ID uint64

// IDGenerator generates shard IDs for given seed
type IDGenerator interface {
	// Generate generates shard ID for each given element in counts
	Generate(seed []byte, counts ...uint64) []ID
}

// NewXORModuloIDGenerator returns shard ID generator which computes IDs by taking xor of seed
// and dividing modulo result
func NewXORModuloIDGenerator() IDGenerator {
	return &xorModuloShardIDGenerator{}
}

type xorModuloShardIDGenerator struct {
}

func (g *xorModuloShardIDGenerator) Generate(seed []byte, counts ...uint64) []ID {
	xor := make([]byte, 8)
	for i, b := range seed {
		xor[i%len(xor)] ^= b
	}
	preID := binary.BigEndian.Uint64(xor)

	results := make([]ID, 0, len(counts))
	for _, c := range counts {
		results = append(results, ID(preID%c))
	}
	return results
}
