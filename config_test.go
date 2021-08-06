package dockergen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWait(t *testing.T) {
	incorrectIntervals := []string{
		"500x",    // Incorrect min interval
		"500s:4x", // Incorrect max interval
		"1m:1s",   // Min interval larger than max interval
	}

	for _, intervalString := range incorrectIntervals {
		wait, err := ParseWait(intervalString)
		assert.Error(t, err)
		assert.Nil(t, wait)
	}

	correctIntervals := map[string]Wait{
		"":          {0, 0},               // Empty time interval string
		"1ms":       {1000000, 4000000},   // Correct min interval without max
		"1ms:111ms": {1000000, 111000000}, // Correct min:max time interval
	}

	for intervalString, expectedWait := range correctIntervals {
		wait, err := ParseWait(intervalString)
		assert.NoError(t, err)
		assert.Equal(t, &expectedWait, wait)
	}
}

func TestWaitUnmarshalText(t *testing.T) {
	// Correct min:max time interval
	intervalBytes := []byte("1ms:2ms")
	expectedWait := &Wait{1000000, 2000000}
	wait := new(Wait)
	err := wait.UnmarshalText(intervalBytes)
	assert.NoError(t, err)
	assert.Equal(t, expectedWait, wait)
}
