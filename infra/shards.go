package infra

import (
	"encoding/binary"
	"strings"

	"github.com/wojciech-malota-wojcik/netdata-digest/infra/wire"
)

// ShardID is the ID of the shard
type ShardID uint64

// DeriveShardID computes shard ID based on userID
// Returned value is in range [0, numOfShards)
func DeriveShardID(userID wire.UserID, numOfShards uint64) ShardID {
	rawShardID := make([]byte, 8)
	for i, b := range []byte(strings.ReplaceAll(string(userID), "-", "")) {
		rawShardID[i%len(rawShardID)] ^= b
	}
	return ShardID(binary.BigEndian.Uint64(rawShardID) % numOfShards)
}
