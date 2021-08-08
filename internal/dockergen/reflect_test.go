package dockergen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeepGetNoPath(t *testing.T) {
	item := RuntimeContainer{}
	value := deepGet(item, "")
	if _, ok := value.(RuntimeContainer); !ok {
		t.Fail()
	}

	returned := value.(RuntimeContainer)
	if !returned.Equals(item) {
		t.Fail()
	}
}

func TestDeepGetSimple(t *testing.T) {
	item := RuntimeContainer{
		ID: "expected",
	}
	value := deepGet(item, "ID")
	assert.IsType(t, "", value)

	assert.Equal(t, "expected", value)
}

func TestDeepGetSimpleDotPrefix(t *testing.T) {
	item := RuntimeContainer{
		ID: "expected",
	}
	value := deepGet(item, "...ID")
	assert.IsType(t, "", value)

	assert.Equal(t, "expected", value)
}

func TestDeepGetMap(t *testing.T) {
	item := RuntimeContainer{
		Env: map[string]string{
			"key": "value",
		},
	}
	value := deepGet(item, "Env.key")
	assert.IsType(t, "", value)

	assert.Equal(t, "value", value)
}
