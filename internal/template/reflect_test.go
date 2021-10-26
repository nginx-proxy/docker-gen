package template

import (
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/stretchr/testify/assert"
)

func TestDeepGetNoPath(t *testing.T) {
	item := context.RuntimeContainer{}
	value := deepGet(item, "")
	if _, ok := value.(context.RuntimeContainer); !ok {
		t.Fail()
	}

	returned := value.(context.RuntimeContainer)
	if !returned.Equals(item) {
		t.Fail()
	}
}

func TestDeepGetSimple(t *testing.T) {
	item := context.RuntimeContainer{
		ID: "expected",
	}
	value := deepGet(item, "ID")
	assert.IsType(t, "", value)

	assert.Equal(t, "expected", value)
}

func TestDeepGetSimpleDotPrefix(t *testing.T) {
	item := context.RuntimeContainer{
		ID: "expected",
	}
	value := deepGet(item, "...ID")
	assert.IsType(t, "", value)

	assert.Equal(t, "expected", value)
}

func TestDeepGetMap(t *testing.T) {
	item := context.RuntimeContainer{
		Env: map[string]string{
			"key": "value",
		},
	}
	value := deepGet(item, "Env.key")
	assert.IsType(t, "", value)

	assert.Equal(t, "value", value)
}
