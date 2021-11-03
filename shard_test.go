package netdata

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wojciech-malota-wojcik/netdata/infra/wire"
	"github.com/wojciech-malota-wojcik/netdata/lib/logger"
)

func runLocalShardTest(t *testing.T, messages ...interface{}) []wire.AlarmDigest {
	ctx, cancel := context.WithCancel(logger.WithLogger(context.Background(), logger.New()))
	defer cancel()
	t.Cleanup(func() {
		cancel()
	})
	responseCapacity := 0
	rx := make(chan interface{}, len(messages))
	for _, msg := range messages {
		rx <- msg
		if _, ok := msg.(wire.SendAlarmDigest); ok {
			responseCapacity++
		}
	}
	close(rx)
	tx := make(chan interface{}, responseCapacity)
	require.NoError(t, runLocalShard(0, rx, tx)(ctx))
	close(tx)

	result := make([]wire.AlarmDigest, 0, responseCapacity)
	for resp := range tx {
		result = append(result, *resp.(*wire.AlarmDigest))
	}
	return result
}

const (
	user1 wire.UserID = "user1"
	user2 wire.UserID = "user2"
	user3 wire.UserID = "user3"

	alarm1 wire.AlarmID = "alarm1"
	alarm2 wire.AlarmID = "alarm2"
	alarm3 wire.AlarmID = "alarm3"
)

var (
	time1 = time.Date(2020, 01, 01, 00, 00, 00, 00, time.UTC)
	time2 = time.Date(2021, 01, 01, 00, 00, 00, 00, time.UTC)
	time3 = time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
	time4 = time.Date(2023, 01, 01, 00, 00, 00, 00, time.UTC)
)

func change(userID wire.UserID, alarmID wire.AlarmID, status wire.Status, time time.Time) wire.AlarmStatusChanged {
	return wire.AlarmStatusChanged{
		ShardedEntity: wire.ShardedEntity{
			UserID: userID,
		},
		AlarmID:   alarmID,
		Status:    status,
		ChangedAt: time,
	}
}

func send(userID wire.UserID) wire.SendAlarmDigest {
	return wire.SendAlarmDigest{
		ShardedEntity: wire.ShardedEntity{
			UserID: userID,
		},
	}
}

func TestNoSendNoDigest(t *testing.T) {
	assert.Len(t, runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		change(user2, alarm2, wire.StatusWarning, time2),
		change(user3, alarm3, wire.StatusCleared, time3),
		change(user1, alarm1, wire.StatusCritical, time4),
	), 0)
}

func TestNoUpdatesNoDigest(t *testing.T) {
	assert.Len(t, runLocalShardTest(t,
		send(user1),
	), 0)
}

func TestSimpleSingleAlarmDigest(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time1,
			},
		},
	}, result[0])
}

func TestFollowingStatusUpdate(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		change(user1, alarm1, wire.StatusWarning, time2),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusWarning,
				LatestChangedAt: time2,
			},
		},
	}, result[0])
}

func TestOutOfOrderStatusUpdate(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time2),
		change(user1, alarm1, wire.StatusWarning, time1),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time2,
			},
		},
	}, result[0])
}

func TestSameStatusTwice(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		change(user1, alarm1, wire.StatusCritical, time2),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time2,
			},
		},
	}, result[0])
}

func TestUpdateBackAndForth(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		change(user1, alarm1, wire.StatusCleared, time2),
		change(user1, alarm1, wire.StatusCritical, time3),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time3,
			},
		},
	}, result[0])
}

func TestNoSecondDigestIfThereWasNoNewUpdate(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time1,
			},
		},
	}, result[0])
}

func TestNoSecondDigestOnRepeatedUpdate(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time1,
			},
		},
	}, result[0])
}

func TestNoSecondDigestIfStatusIsTheSame(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
		change(user1, alarm1, wire.StatusCritical, time2),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time1,
			},
		},
	}, result[0])
}

func TestNoSecondDigestIfStatusIsTheSameAndAnotherUpdateComesOutOfOrder(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
		change(user1, alarm1, wire.StatusCritical, time3),
		change(user1, alarm1, wire.StatusWarning, time2),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time1,
			},
		},
	}, result[0])
}

func TestNoSecondDigestIfStatusIsCleared(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
		change(user1, alarm1, wire.StatusCleared, time2),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time1,
			},
		},
	}, result[0])
}

func TestNoSecondDigestIfUpdateIsOutOfOrder(t *testing.T) {
	result := runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time2),
		send(user1),
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
	)
	require.Len(t, result, 1)
	assert.Equal(t, wire.AlarmDigest{
		UserID: user1,
		ActiveAlarms: []wire.Alarm{
			{
				AlarmID:         alarm1,
				Status:          wire.StatusCritical,
				LatestChangedAt: time2,
			},
		},
	}, result[0])
}

func TestNoDigestIfAlarmIsCleared(t *testing.T) {
	require.Len(t, runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCritical, time1),
		change(user1, alarm1, wire.StatusCleared, time2),
		send(user1),
	), 0)
}

func TestNoDigestIfAlarmIsClearedOutOfOrder(t *testing.T) {
	require.Len(t, runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCleared, time2),
		change(user1, alarm1, wire.StatusCritical, time1),
		send(user1),
	), 0)
}

func TestNoDigestIfAlarmIsClearedWithOutOfOrderUpdateInBetween(t *testing.T) {
	require.Len(t, runLocalShardTest(t,
		change(user1, alarm1, wire.StatusCleared, time1),
		change(user1, alarm1, wire.StatusCleared, time3),
		change(user1, alarm1, wire.StatusCritical, time2),
		send(user1),
	), 0)
}
