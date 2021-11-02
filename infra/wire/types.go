package wire

import (
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

// VerifyStatus verifies that incoming status is one of accepted values
func VerifyStatus(status Status) error {
	switch status {
	case StatusCleared, StatusWarning, StatusCritical:
		return nil
	default:
		return fmt.Errorf("unknown status: %s", status)
	}
}

// AlarmStatusChanged is the incoming AlarmStatusChanged message
type AlarmStatusChanged struct {
	// UserID is the user ID
	UserID UserID

	// AlarmID is the alarm ID
	AlarmID AlarmID

	// Status is the status of the alarm
	Status Status

	// ChangedAt is the time when the status changed
	ChangedAt time.Time
}

// SendAlarmDigest is the incoming SendAlarmDigest message
type SendAlarmDigest struct {
	// UserID is the user ID
	UserID UserID
}

// Alarm contains the current state of the alarm
type Alarm struct {
	// AlarmID is the alarm ID
	AlarmID AlarmID

	// Status is the last reported status
	Status Status

	// LastChangedAt is the time when status was updated
	LastChangedAt time.Time
}

// AlarmDigest is the outgoing AlarmDigest message
type AlarmDigest struct {
	// UserID is the UserID
	UserID UserID

	// ActiveAlarms is the list of active alarms
	ActiveAlarms []Alarm
}
