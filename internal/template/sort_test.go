package template

import (
	"testing"
	"time"

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

func TestGetFieldAsString(t *testing.T) {
	testStruct := struct {
		String string
		BoolT  bool
		BoolF  bool
		Int    int
		Int32  int32
		Int64  int64
		Time   time.Time
	}{
		String: "foo",
		BoolT:  true,
		BoolF:  false,
		Int:    42,
		Int32:  43,
		Int64:  44,
		Time:   time.Date(2023, 12, 19, 0, 0, 0, 0, time.UTC),
	}

	assert.Equal(t, "foo", getFieldAsString(testStruct, "String"))
	assert.Equal(t, "true", getFieldAsString(testStruct, "BoolT"))
	assert.Equal(t, "false", getFieldAsString(testStruct, "BoolF"))
	assert.Equal(t, "42", getFieldAsString(testStruct, "Int"))
	assert.Equal(t, "43", getFieldAsString(testStruct, "Int32"))
	assert.Equal(t, "44", getFieldAsString(testStruct, "Int64"))
	assert.Equal(t, "2023-12-19 00:00:00 +0000 UTC", getFieldAsString(testStruct, "Time"))
	assert.Equal(t, "", getFieldAsString(testStruct, "InvalidField"))
}

func TestSortObjectsByKeys(t *testing.T) {
	o0 := &context.RuntimeContainer{
		Created: time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
		Env: map[string]string{
			"VIRTUAL_HOST": "bar.localhost",
		},
		Labels: map[string]string{
			"com.docker.compose.container_number": "1",
		},
		ID: "11",
	}
	o1 := &context.RuntimeContainer{
		Created: time.Date(2021, 1, 2, 0, 0, 10, 0, time.UTC),
		Env: map[string]string{
			"VIRTUAL_HOST": "foo.localhost",
		},
		Labels: map[string]string{
			"com.docker.compose.container_number": "11",
		},
		ID: "1",
	}
	o2 := &context.RuntimeContainer{
		Created: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		Env: map[string]string{
			"VIRTUAL_HOST": "baz.localhost",
		},
		Labels: map[string]string{},
		ID:     "3",
	}
	o3 := &context.RuntimeContainer{
		Created: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Env:     map[string]string{},
		Labels: map[string]string{
			"com.docker.compose.container_number": "2",
		},
		ID: "8",
	}
	containers := []*context.RuntimeContainer{o0, o1, o2, o3}

	for _, tc := range []struct {
		desc string
		fn   func(interface{}, string) ([]interface{}, error)
		key  string
		want []interface{}
	}{
		{"Asc simple", sortObjectsByKeysAsc, "ID", []interface{}{o1, o2, o3, o0}},
		{"Desc simple", sortObjectsByKeysDesc, "ID", []interface{}{o0, o3, o2, o1}},
		{"Asc complex", sortObjectsByKeysAsc, "Env.VIRTUAL_HOST", []interface{}{o3, o0, o2, o1}},
		{"Desc complex", sortObjectsByKeysDesc, "Env.VIRTUAL_HOST", []interface{}{o1, o2, o0, o3}},
		{"Asc complex w/ dots in key name", sortObjectsByKeysAsc, "Labels.com.docker.compose.container_number", []interface{}{o2, o0, o3, o1}},
		{"Desc complex w/ dots in key name", sortObjectsByKeysDesc, "Labels.com.docker.compose.container_number", []interface{}{o1, o3, o0, o2}},
		{"Asc time", sortObjectsByKeysAsc, "Created", []interface{}{o3, o0, o2, o1}},
		{"Desc time", sortObjectsByKeysDesc, "Created", []interface{}{o1, o2, o0, o3}},
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
