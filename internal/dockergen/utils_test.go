package dockergen

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDefaultEndpoint(t *testing.T) {
	err := os.Unsetenv("DOCKER_HOST")
	if err != nil {
		t.Fatalf("Unable to unset DOCKER_HOST: %s", err)
	}

	endpoint, err := GetEndpoint("")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if endpoint != "unix:///var/run/docker.sock" {
		t.Fatalf("Expected unix:///var/run/docker.sock, got %s", endpoint)
	}
}

func TestDockerHostEndpoint(t *testing.T) {
	err := os.Setenv("DOCKER_HOST", "tcp://127.0.0.1:4243")
	if err != nil {
		t.Fatalf("Unable to set DOCKER_HOST: %s", err)
	}

	endpoint, err := GetEndpoint("")
	if err != nil {
		t.Fatalf("%s", err)
	}

	if endpoint != "tcp://127.0.0.1:4243" {
		t.Fatalf("Expected tcp://127.0.0.1:4243, got %s", endpoint)
	}
}

func TestDockerFlagEndpoint(t *testing.T) {

	err := os.Setenv("DOCKER_HOST", "tcp://127.0.0.1:4243")
	if err != nil {
		t.Fatalf("Unable to set DOCKER_HOST: %s", err)
	}

	// flag value should override DOCKER_HOST and default value
	endpoint, err := GetEndpoint("tcp://127.0.0.1:5555")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if endpoint != "tcp://127.0.0.1:5555" {
		t.Fatalf("Expected tcp://127.0.0.1:5555, got %s", endpoint)
	}
}

func TestUnixBadFormat(t *testing.T) {
	endpoint := "unix:/var/run/docker.sock"
	_, err := GetEndpoint(endpoint)
	if err == nil {
		t.Fatal("endpoint should have failed")
	}
}

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
		v := splitKeyValueSlice(i.input)
		if v["K"] != i.expected {
			t.Fatalf("expected K='%s'. got '%s'", i.expected, v["K"])
		}

	}
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{" ", true},
		{"   ", true},
		{"\t", true},
		{"\t\n\v\f\r\u0085\u00A0", true},
		{"a", false},
		{" a ", false},
		{"a ", false},
		{" a", false},
		{"日本語", false},
	}

	for _, i := range tests {
		v := isBlank(i.input)
		if v != i.expected {
			t.Fatalf("expected '%v'. got '%v'", i.expected, v)
		}
	}
}

func TestRemoveBlankLines(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\r\n\r\n", ""},
		{"line1\nline2", "line1\nline2"},
		{"line1\n\nline2", "line1\nline2"},
		{"\n\n\n\nline1\n\nline2", "line1\nline2"},
		{"\n\n\n\n\n  \n \n \n", ""},

		// windows line endings \r\n
		{"line1\r\nline2", "line1\r\nline2"},
		{"line1\r\n\r\nline2", "line1\r\nline2"},

		// keep last new line
		{"line1\n", "line1\n"},
		{"line1\r\n", "line1\r\n"},
	}

	for _, i := range tests {
		output := new(bytes.Buffer)
		removeBlankLines(strings.NewReader(i.input), output)
		if output.String() != i.expected {
			t.Fatalf("expected '%v'. got '%v'", i.expected, output)
		}
	}
}
