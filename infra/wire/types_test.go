package wire

import (
	"testing"

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
