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

func TestSortObjectsByKeysAsc(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar.localhost",
			},
			ID: "9",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "foo.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "baz.localhost",
			},
			ID: "3",
		},
		{
			Env: map[string]string{},
			ID:  "8",
		},
	}

	sorted, err := sortObjectsByKeysAsc(containers, "ID")

	assert.NoError(t, err)
	assert.Len(t, sorted, 4)
	assert.Equal(t, "foo.localhost", sorted[0].(*context.RuntimeContainer).Env["VIRTUAL_HOST"])
	assert.Equal(t, "9", sorted[3].(*context.RuntimeContainer).ID)

	sorted, err = sortObjectsByKeysAsc(sorted, "Env.VIRTUAL_HOST")

	assert.NoError(t, err)
	assert.Len(t, sorted, 4)
	assert.Equal(t, "foo.localhost", sorted[3].(*context.RuntimeContainer).Env["VIRTUAL_HOST"])
	assert.Equal(t, "8", sorted[0].(*context.RuntimeContainer).ID)
}

func TestSortObjectsByKeysDesc(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar.localhost",
			},
			ID: "9",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "foo.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "baz.localhost",
			},
			ID: "3",
		},
		{
			Env: map[string]string{},
			ID:  "8",
		},
	}

	sorted, err := sortObjectsByKeysDesc(containers, "ID")

	assert.NoError(t, err)
	assert.Len(t, sorted, 4)
	assert.Equal(t, "bar.localhost", sorted[0].(*context.RuntimeContainer).Env["VIRTUAL_HOST"])
	assert.Equal(t, "1", sorted[3].(*context.RuntimeContainer).ID)

	sorted, err = sortObjectsByKeysDesc(sorted, "Env.VIRTUAL_HOST")

	assert.NoError(t, err)
	assert.Len(t, sorted, 4)
	assert.Equal(t, "", sorted[3].(*context.RuntimeContainer).Env["VIRTUAL_HOST"])
	assert.Equal(t, "1", sorted[0].(*context.RuntimeContainer).ID)
}
