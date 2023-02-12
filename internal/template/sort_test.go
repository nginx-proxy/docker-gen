package template

import (
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/stretchr/testify/assert"
)

func TestSortStringsAsc(t *testing.T) {
	strings := []string{"foo", "bar", "baz", "qux"}
	expected := []string{"bar", "baz", "foo", "qux"}
	assert.Equal(t, expected, sortStringsAsc(strings))
}

func TestSortStringsDesc(t *testing.T) {
	strings := []string{"foo", "bar", "baz", "qux"}
	expected := []string{"qux", "foo", "baz", "bar"}
	assert.Equal(t, expected, sortStringsDesc(strings))
}

func TestSortObjectsByKeys(t *testing.T) {
	o0 := &context.RuntimeContainer{
		Env: map[string]string{
			"VIRTUAL_HOST": "bar.localhost",
		},
		ID: "9",
	}
	o1 := &context.RuntimeContainer{
		Env: map[string]string{
			"VIRTUAL_HOST": "foo.localhost",
		},
		ID: "1",
	}
	o2 := &context.RuntimeContainer{
		Env: map[string]string{
			"VIRTUAL_HOST": "baz.localhost",
		},
		ID: "3",
	}
	o3 := &context.RuntimeContainer{
		Env: map[string]string{},
		ID:  "8",
	}
	containers := []*context.RuntimeContainer{o0, o1, o2, o3}

	for _, tc := range []struct {
		desc string
		fn   func(interface{}, string) ([]interface{}, error)
		key  string
		want []interface{}
	}{
		{"Asc simple", sortObjectsByKeysAsc, "ID", []interface{}{o1, o2, o3, o0}},
		{"Asc complex", sortObjectsByKeysAsc, "Env.VIRTUAL_HOST", []interface{}{o3, o0, o2, o1}},
		{"Desc simple", sortObjectsByKeysDesc, "ID", []interface{}{o0, o3, o2, o1}},
		{"Desc complex", sortObjectsByKeysDesc, "Env.VIRTUAL_HOST", []interface{}{o1, o2, o0, o3}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.fn(containers, tc.key)
			assert.NoError(t, err)
			// The function should return a sorted copy of the slice, not modify the original.
			assert.Equal(t, []*context.RuntimeContainer{o0, o1, o2, o3}, containers)
			assert.Equal(t, tc.want, got)
		})
	}
}
