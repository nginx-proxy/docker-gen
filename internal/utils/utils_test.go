package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitKeyValueSlice(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{"K"}, ""},
		{[]string{"K="}, ""},
		{[]string{"K=V3"}, "V3"},
		{[]string{"K=V4=V5"}, "V4=V5"},
	}

	for _, i := range tests {
		v := SplitKeyValueSlice(i.input)
		if v["K"] != i.expected {
			t.Fatalf("expected K='%s'. got '%s'", i.expected, v["K"])
		}

	}
}

func TestPathExists(t *testing.T) {
	file, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	exists, err := PathExists(file.Name())
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = PathExists("/wrong/path")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestPathLExists(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// Never created, so it is guaranteed not to exist on any platform.
	missing := filepath.Join(dir, "missing")

	ok, err := PathLExists(file)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = PathLExists(missing)
	assert.NoError(t, err)
	assert.False(t, ok)

	link := filepath.Join(dir, "link")
	if err := os.Symlink(missing, link); err != nil {
		t.Skipf("symlinks unavailable: %s", err)
	}

	ok, err = PathLExists(link)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = PathExists(link)
	assert.NoError(t, err)
	assert.False(t, ok)
}
