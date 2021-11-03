package bus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type someEntity struct {
}

func TestTypeToTopic(t *testing.T) {
	assert.Equal(t, "someEntity", topicForValue(&someEntity{}))
}
