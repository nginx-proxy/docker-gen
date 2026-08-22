package utils

import (
	"os"
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
	file, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	ok, err := PathLExists(file.Name())
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = PathLExists("/wrong/path")
	assert.NoError(t, err)
	assert.False(t, ok)

	link := file.Name() + "-link"
	if err := os.Symlink("/wrong/path", link); err != nil {
		t.Skipf("symlinks unavailable: %s", err)
	}
	defer os.Remove(link)

	ok, err = PathLExists(link)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = PathExists(link)
	assert.NoError(t, err)
	assert.False(t, ok)
}
