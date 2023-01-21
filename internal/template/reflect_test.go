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
	value := deepGet(item, ".ID")
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

func TestDeepGet(t *testing.T) {
	s := struct{ X string }{"foo"}
	sp := &s

	for _, tc := range []struct {
		desc string
		item interface{}
		path string
		want interface{}
	}{
		{
			"map key empty string",
			map[string]map[string]map[string]string{
				"": map[string]map[string]string{
					"": map[string]string{
						"": "foo",
					},
				},
			},
			"...",
			"foo",
		},
		{"struct", s, "X", "foo"},
		{"pointer to struct", sp, "X", "foo"},
		{"double pointer to struct", &sp, ".X", nil},
		{"slice index", []string{"foo", "bar"}, "1", "bar"},
		{"slice index out of bounds", []string{}, "0", nil},
		{"slice index negative", []string{}, "-1", nil},
		{"slice index nonnumber", []string{}, "foo", nil},
		{"array index", [2]string{"foo", "bar"}, "1", "bar"},
		{"array index out of bounds", [1]string{"foo"}, "1", nil},
		{"array index negative", [1]string{"foo"}, "-1", nil},
		{"array index nonnumber", [1]string{"foo"}, "foo", nil},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := deepGet(tc.item, tc.path)
			assert.IsType(t, tc.want, got)
			assert.Equal(t, tc.want, got)
		})
	}
}
