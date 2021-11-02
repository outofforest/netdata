package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCleared(t *testing.T) {
	assert.NoError(t, VerifyStatus(Status("CLEARED")))
}

func TestStatusWarning(t *testing.T) {
	assert.NoError(t, VerifyStatus(Status("WARNING")))
}

func TestStatusCritical(t *testing.T) {
	assert.NoError(t, VerifyStatus(Status("CRITICAL")))
}

func TestStatusIncorrect1(t *testing.T) {
	assert.Error(t, VerifyStatus(Status("cleared")))
}

func TestStatusIncorrect2(t *testing.T) {
	assert.Error(t, VerifyStatus(Status("warning")))
}

func TestStatusIncorrect3(t *testing.T) {
	assert.Error(t, VerifyStatus(Status("critical")))
}

func TestStatusIncorrect4(t *testing.T) {
	assert.Error(t, VerifyStatus(Status("weird")))
}
