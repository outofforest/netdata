package bus

import "encoding/binary"

// deriveShardPreID computes shard ID based on userID
func deriveShardPreID(shardSeed []byte) uint64 {
	rawPreShardID := make([]byte, 8)
	for i, b := range shardSeed {
		rawPreShardID[i%len(rawPreShardID)] ^= b
	}
	return binary.BigEndian.Uint64(rawPreShardID)
}
