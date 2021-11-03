package wire

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatusCleared(t *testing.T) {
	assert.NoError(t, verifyStatus(Status("CLEARED")))
}

func TestStatusWarning(t *testing.T) {
	assert.NoError(t, verifyStatus(Status("WARNING")))
}

func TestStatusCritical(t *testing.T) {
	assert.NoError(t, verifyStatus(Status("CRITICAL")))
}

func TestStatusIncorrect1(t *testing.T) {
	assert.Error(t, verifyStatus(Status("cleared")))
}

func TestStatusIncorrect2(t *testing.T) {
	assert.Error(t, verifyStatus(Status("warning")))
}

func TestStatusIncorrect3(t *testing.T) {
	assert.Error(t, verifyStatus(Status("critical")))
}

func TestStatusIncorrect4(t *testing.T) {
	assert.Error(t, verifyStatus(Status("weird")))
}

func TestStatusIncorrect5(t *testing.T) {
	assert.Error(t, verifyStatus(Status("")))
}

func TestValidateAlarmStatusChanged(t *testing.T) {
	entity := AlarmStatusChanged{
		ShardedEntity: ShardedEntity{
			UserID: "userID",
		},
		AlarmID:   "alarmID",
		Status:    StatusCleared,
		ChangedAt: time.Now(),
	}
	assert.NoError(t, entity.Validate())

	e := entity
	e.UserID = ""
	assert.Error(t, e.Validate())

	e = entity
	e.AlarmID = ""
	assert.Error(t, e.Validate())

	e = entity
	e.Status = "invalid"
	assert.Error(t, e.Validate())

	e = entity
	e.ChangedAt = time.Time{}
	assert.Error(t, e.Validate())
}

func TestValidateSendAlarmDigest(t *testing.T) {
	entity := SendAlarmDigest{
		ShardedEntity: ShardedEntity{
			UserID: "userID",
		},
	}
	assert.NoError(t, entity.Validate())

	e := entity
	e.UserID = ""
	assert.Error(t, e.Validate())
}

func TestShardSeedAlarmStatusChanged(t *testing.T) {
	entity := AlarmStatusChanged{
		ShardedEntity: ShardedEntity{
			UserID: "userID",
		},
		AlarmID:   "alarmID",
		Status:    StatusCleared,
		ChangedAt: time.Now(),
	}
	assert.Equal(t, []byte("userID"), entity.ShardSeed())
}

func TestShardSeedSendAlarmDigest(t *testing.T) {
	entity := SendAlarmDigest{
		ShardedEntity: ShardedEntity{
			UserID: "userID",
		},
	}
	assert.Equal(t, []byte("userID"), entity.ShardSeed())
}
