package bus

import (
	"encoding/json"
	"reflect"

	"github.com/wojciech-malota-wojcik/netdata/infra"
	"github.com/wojciech-malota-wojcik/netdata/infra/sharding"
	"go.uber.org/zap"
)

// NewDispatcherFactory creates new dispatcher factory
func NewDispatcherFactory(config infra.Config, shardIDGen sharding.IDGenerator) DispatcherFactory {
	return &dispatcherFactory{
		config:     config,
		shardIDGen: shardIDGen,
	}
}

type dispatcherFactory struct {
	config     infra.Config
	shardIDGen sharding.IDGenerator
}

func (df *dispatcherFactory) Create(templatePtr Entity, recvChs []chan<- interface{}, log *zap.Logger) Dispatcher {
	return &dispatcher{
		config:     df.config,
		shardIDGen: df.shardIDGen,
		log:        log,

		templatePtr:   templatePtr,
		templateValue: reflect.ValueOf(templatePtr).Elem(),
		recvChs:       recvChs,
	}
}

type dispatcher struct {
	config     infra.Config
	shardIDGen sharding.IDGenerator
	log        *zap.Logger

	templatePtr   Entity
	templateValue reflect.Value
	recvChs       []chan<- interface{}
}

func (d *dispatcher) Dispatch(msg []byte) {
	d.log.Debug("Message received", zap.ByteString("msg", msg))
	if err := json.Unmarshal(msg, d.templatePtr); err != nil {
		d.log.Error("Decoding message failed", zap.Error(err))
		return
	}
	if err := d.templatePtr.Validate(); err != nil {
		d.log.Error("Received entity is in invalid state", zap.Error(err))
		return
	}
	shardIDs := d.shardIDGen.Generate(d.templatePtr.ShardSeed(), d.config.NumOfShards, uint64(len(d.recvChs)))
	if shardID := shardIDs[0]; shardID != d.config.ShardID {
		d.log.Debug("Entity not for this shard received, ignoring", zap.Any("dstShardID", shardID), zap.Any("shardID", d.config.ShardID))
		return
	}

	localShardID := shardIDs[1]
	d.recvChs[localShardID] <- d.templateValue.Interface()
}
