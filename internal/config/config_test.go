package config

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestFilterWatches(t *testing.T) {
	testConfigFile := &ConfigFile{
		Config: []Config{
			{Template: "foo", Watch: true},
			{Template: "bar"},
			{Template: "baz", Watch: true},
		},
	}

	expected := []Config{
		{Template: "foo", Watch: true},
		{Template: "baz", Watch: true},
	}

	configFile := testConfigFile.FilterWatches()
	assert.Equal(t, expected, configFile.Config)
}

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

func TestSetContainerFilterDefaults(t *testing.T) {
	allStatuses := []string{"created", "restarting", "running", "removing", "paused", "exited", "dead"}

	tests := []struct {
		name     string
		config   Config
		expected map[string][]string
	}{
		{
			name:     "no options",
			config:   Config{},
			expected: map[string][]string{"status": {"running"}},
		},
		{
			name:     "include stopped",
			config:   Config{IncludeStopped: true},
			expected: map[string][]string{"status": allStatuses},
		},
		{
			name:     "status filter",
			config:   Config{IncludeStopped: true, ContainerFilter: map[string][]string{"status": {"exited"}}},
			expected: map[string][]string{"status": {"exited"}},
		},
		{
			name:     "only exposed",
			config:   Config{OnlyExposed: true},
			expected: map[string][]string{"status": {"running"}, "expose": {"1-65535"}},
		},
		{
			name:     "expose filter",
			config:   Config{OnlyExposed: true, ContainerFilter: map[string][]string{"expose": {"80"}}},
			expected: map[string][]string{"status": {"running"}, "expose": {"80"}},
		},
		{
			name:     "only published",
			config:   Config{OnlyExposed: true, OnlyPublished: true},
			expected: map[string][]string{"status": {"running"}, "publish": {"1-65535"}},
		},
		{
			name:     "publish filter",
			config:   Config{OnlyPublished: true, ContainerFilter: map[string][]string{"publish": {"443"}}},
			expected: map[string][]string{"status": {"running"}, "publish": {"443"}},
		},
		{
			name:     "other filter",
			config:   Config{ContainerFilter: map[string][]string{"label": {"foo"}}},
			expected: map[string][]string{"status": {"running"}, "label": {"foo"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.config.SetContainerFilterDefaults()
			assert.Equal(t, test.expected, test.config.ContainerFilter)
		})
	}
}

func TestConfigFileContainerFilter(t *testing.T) {
	var configFile ConfigFile
	_, err := toml.Decode(`
[[config]]
template = "exposed.tmpl"
onlyexposed = true

[[config]]
template = "published.tmpl"
onlypublished = true
includestopped = true

[[config]]
template = "filtered.tmpl"
onlyexposed = true

[config.ContainerFilter]
status = ["running", "paused"]
label = ["foo"]
`, &configFile)
	assert.NoError(t, err)

	for i := range configFile.Config {
		configFile.Config[i].SetContainerFilterDefaults()
	}

	assert.Len(t, configFile.Config, 3)
	assert.Equal(t, map[string][]string{"status": {"running"}, "expose": {"1-65535"}}, configFile.Config[0].ContainerFilter)
	assert.Equal(t, map[string][]string{
		"status":  {"created", "restarting", "running", "removing", "paused", "exited", "dead"},
		"publish": {"1-65535"},
	}, configFile.Config[1].ContainerFilter)
	assert.Equal(t, map[string][]string{
		"status": {"running", "paused"},
		"label":  {"foo"},
		"expose": {"1-65535"},
	}, configFile.Config[2].ContainerFilter)
}
