package dockergen

import (
	"testing"
)

func TestGetCurrentContainerID(t *testing.T) {
	currentContainerID := GetCurrentContainerID()

	if len(currentContainerID) != 64 {
		t.Fail()
	}
}
