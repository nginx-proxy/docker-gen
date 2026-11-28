package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetGlobals(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		configs = config.ConfigFile{}
		containerFilter = make(mapstringslice)
		notifyContainerFilter = make(mapstringslice)
		notifyContainerSignal = 0
		includeStopped = false
		onlyExposed = false
		onlyPublished = false
	})
}

func TestApplyCLIFiltersToLoadedConfig(t *testing.T) {
	resetGlobals(t)

	cfgPath := filepath.Join(t.TempDir(), "docker-gen.cfg")
	cfgContent := `
[[config]]
template = "/etc/docker-gen/templates/a.tmpl"
dest = "/etc/nginx/conf.d/a.conf"

[[config]]
template = "/etc/docker-gen/templates/b.tmpl"
dest = "/etc/nginx/conf.d/b.conf"
[config.ContainerFilter]
label = ["custom=1"]
status = ["running", "exited"]
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0o600))
	require.NoError(t, loadConfig(cfgPath))

	require.NoError(t, containerFilter.Set("label=mylabel"))
	require.NoError(t, notifyContainerFilter.Set("name=nginx-proxy"))
	notifyContainerSignal = 1
	onlyExposed = true

	for i := range configs.Config {
		applyCLIFilters(&configs.Config[i])
	}

	require.Len(t, configs.Config, 2)

	// Entry without explicit ContainerFilter inherits CLI/env filters and defaults.
	assert.Equal(t, map[string][]string{
		"label":  {"mylabel"},
		"status": {"running"},
		"expose": {"1-65535"},
	}, configs.Config[0].ContainerFilter)
	assert.Equal(t, map[string][]string{
		"name": {"nginx-proxy"},
	}, configs.Config[0].NotifyContainersFilter)
	assert.Equal(t, 1, configs.Config[0].NotifyContainersSignal)

	// Entry with explicit ContainerFilter preserves its keys while receiving unset flags.
	assert.Equal(t, map[string][]string{
		"label":  {"custom=1"},
		"status": {"running", "exited"},
		"expose": {"1-65535"},
	}, configs.Config[1].ContainerFilter)
}

func TestLoadConfigLowercaseContainerFilter(t *testing.T) {
	resetGlobals(t)

	cfgPath := filepath.Join(t.TempDir(), "docker-gen.cfg")
	cfgContent := `
[[config]]
template = "/etc/docker-gen/templates/a.tmpl"
dest = "/etc/nginx/conf.d/a.conf"
[config.containerfilter]
label = ["mylabel"]
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfgContent), 0o600))
	require.NoError(t, loadConfig(cfgPath))
	require.Len(t, configs.Config, 1)

	includeStopped = true
	applyCLIFilters(&configs.Config[0])

	assert.Equal(t, []string{"mylabel"}, configs.Config[0].ContainerFilter["label"])
	assert.Len(t, configs.Config[0].ContainerFilter["status"], 7)
}
