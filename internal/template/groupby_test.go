package template

import (
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/stretchr/testify/assert"
)

var groupByContainers = []*context.RuntimeContainer{
	{
		Env: map[string]string{
			"VIRTUAL_HOST": "demo1.localhost",
			"EXTERNAL":     "true",
		},
		ID: "1",
	},
	{
		Env: map[string]string{
			"VIRTUAL_HOST": "demo1.localhost",
		},
		ID: "2",
	},
	{
		Env: map[string]string{
			"VIRTUAL_HOST": "demo2.localhost",
			"EXTERNAL":     "true",
		},
		ID: "3",
	},
	{
		ID: "4",
	},
}

func TestGroupByExistingKey(t *testing.T) {
	groups, err := groupBy(groupByContainers, "Env.VIRTUAL_HOST")

	assert.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Len(t, groups["demo1.localhost"], 2)
	assert.Len(t, groups["demo2.localhost"], 1)
	assert.Equal(t, "3", groups["demo2.localhost"][0].(*context.RuntimeContainer).ID)
}

func TestGroupByAfterWhere(t *testing.T) {
	filtered, _ := where(groupByContainers, "Env.EXTERNAL", "true")
	groups, err := groupBy(filtered, "Env.VIRTUAL_HOST")

	assert.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Len(t, groups["demo1.localhost"], 1)
	assert.Len(t, groups["demo2.localhost"], 1)
	assert.Equal(t, "3", groups["demo2.localhost"][0].(*context.RuntimeContainer).ID)
}

func TestGroupByWithDefault(t *testing.T) {
	groups, err := groupByWithDefault(groupByContainers, "Env.VIRTUAL_HOST", "default.localhost")

	assert.NoError(t, err)
	assert.Len(t, groups, 3)
	assert.Len(t, groups["demo1.localhost"], 2)
	assert.Len(t, groups["demo2.localhost"], 1)
	assert.Len(t, groups["default.localhost"], 1)
	assert.Equal(t, "4", groups["default.localhost"][0].(*context.RuntimeContainer).ID)
}

func TestGroupByKeys(t *testing.T) {
	expected := []string{"demo1.localhost", "demo2.localhost"}
	groups, err := groupByKeys(groupByContainers, "Env.VIRTUAL_HOST")
	assert.NoError(t, err)
	assert.ElementsMatch(t, expected, groups)

	expected = []string{"1", "2", "3", "4"}
	groups, err = groupByKeys(groupByContainers, "ID")
	assert.NoError(t, err)
	assert.ElementsMatch(t, expected, groups)
}

func TestGeneralizedGroupByError(t *testing.T) {
	groups, err := groupBy("string", "")
	assert.Error(t, err)
	assert.Nil(t, groups)
}

var groupByLabelContainers = []*context.RuntimeContainer{
	{
		Labels: map[string]string{
			"com.docker.compose.project": "one",
		},
		ID: "1",
	},
	{
		Labels: map[string]string{
			"com.docker.compose.project": "two",
		},
		ID: "2",
	},
	{
		Labels: map[string]string{
			"com.docker.compose.project": "one",
		},
		ID: "3",
	},
	{
		ID: "4",
	},
	{
		Labels: map[string]string{
			"com.docker.compose.project": "",
		},
		ID: "5",
	},
}

func TestGroupByLabel(t *testing.T) {
	groups, err := groupByLabel(groupByLabelContainers, "com.docker.compose.project")

	assert.NoError(t, err)
	assert.Len(t, groups, 3)
	assert.Len(t, groups["one"], 2)
	assert.Len(t, groups[""], 1)
	assert.Len(t, groups["two"], 1)
	assert.Equal(t, "2", groups["two"][0].(*context.RuntimeContainer).ID)
}

func TestGroupByLabelError(t *testing.T) {
	strings := []string{"foo", "bar", "baz"}
	groups, err := groupByLabel(strings, "")
	assert.Error(t, err)
	assert.Nil(t, groups)
}

func TestGroupByLabelWithDefault(t *testing.T) {
	groups, err := groupByLabelWithDefault(groupByLabelContainers, "com.docker.compose.project", "default")

	assert.NoError(t, err)
	assert.Len(t, groups, 4)
	assert.Len(t, groups["one"], 2)
	assert.Len(t, groups["two"], 1)
	assert.Len(t, groups[""], 1)
	assert.Len(t, groups["default"], 1)
	assert.Equal(t, "4", groups["default"][0].(*context.RuntimeContainer).ID)
}

func TestGroupByLabelWithDefaultError(t *testing.T) {
	strings := []string{"foo", "bar", "baz"}
	groups, err := groupByLabelWithDefault(strings, "", "")
	assert.Error(t, err)
	assert.Nil(t, groups)
}

func TestGroupByMulti(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups, _ := groupByMulti(containers, "Env.VIRTUAL_HOST", ",")
	if len(groups) != 3 {
		t.Fatalf("expected 3 got %d", len(groups))
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fatalf("expected 2 got %d", len(groups["demo1.localhost"]))
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.Fatalf("expected 1 got %d", len(groups["demo2.localhost"]))
	}
	if groups["demo2.localhost"][0].(*context.RuntimeContainer).ID != "3" {
		t.Fatalf("expected 2 got %s", groups["demo2.localhost"][0].(*context.RuntimeContainer).ID)
	}
	if len(groups["demo3.localhost"]) != 1 {
		t.Fatalf("expect 1 got %d", len(groups["demo3.localhost"]))
	}
	if groups["demo3.localhost"][0].(*context.RuntimeContainer).ID != "2" {
		t.Fatalf("expected 2 got %s", groups["demo3.localhost"][0].(*context.RuntimeContainer).ID)
	}
}
