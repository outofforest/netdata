package wire

import (
	"errors"
	"fmt"
	"time"
)

// UserID is the user ID
type UserID string

// AlarmID is the alarm ID
type AlarmID string

// Status is the status of alarm
type Status string

const (
	// StatusCleared means there is no alarm
	StatusCleared Status = "CLEARED"

	// StatusWarning means there is a warning for an alarm
	StatusWarning Status = "WARNING"

	// StatusCritical means alarm is in critical state
	StatusCritical Status = "CRITICAL"
)

// ShardedEntity is data entity which is a subject of sharding
type ShardedEntity struct {
	// UserID is the user ID
	UserID UserID
}

// ShardSeed generates a seed used to compute shard ID
func (e *ShardedEntity) ShardSeed() []byte {
	return []byte(e.UserID)
}

// AlarmStatusChanged is the incoming AlarmStatusChanged message
type AlarmStatusChanged struct {
	ShardedEntity

	// AlarmID is the alarm ID
	AlarmID AlarmID

	// Status is the status of the alarm
	Status Status

	// ChangedAt is the time when the status changed
	ChangedAt time.Time
}

// Validate validates if message contains valid data
func (o AlarmStatusChanged) Validate() error {
	if o.UserID == "" {
		return errors.New("field UserID is empty")
	}
	if o.AlarmID == "" {
		return errors.New("field AlarmID is empty")
	}
	if o.ChangedAt.IsZero() {
		return errors.New("field ChangedAt is a zero time")
	}
	return verifyStatus(o.Status)
}

// SendAlarmDigest is the incoming SendAlarmDigest message
type SendAlarmDigest struct {
	ShardedEntity
}

// Validate validates if message contains valid data
func (o SendAlarmDigest) Validate() error {
	if o.UserID == "" {
		return errors.New("field UserID is empty")
	}
	return nil
}

// Alarm contains the current state of the alarm
type Alarm struct {
	// AlarmID is the alarm ID
	AlarmID AlarmID

	// Status is the last reported status
	Status Status

	// LatestChangedAt is the time when status was updated
	LatestChangedAt time.Time
}

// AlarmDigest is the outgoing AlarmDigest message
type AlarmDigest struct {
	// UserID is the UserID
	UserID UserID

	// ActiveAlarms is the list of active alarms
	ActiveAlarms []Alarm
}

// verifyStatus verifies that incoming status is one of accepted values
func verifyStatus(status Status) error {
	switch status {
	case StatusCleared, StatusWarning, StatusCritical:
		return nil
	case "":
		return errors.New("status is empty")
	default:
		return fmt.Errorf("unknown status: %s", status)
	}
}
