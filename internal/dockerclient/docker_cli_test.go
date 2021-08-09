package dockerclient

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/stretchr/testify/assert"
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

func TestSplitDockerImageRepository(t *testing.T) {
	registry, repository, tag := SplitDockerImage("ubuntu")

	assert.Equal(t, "", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "", tag)

	dockerImage := context.DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "ubuntu", dockerImage.String())
}

func TestSplitDockerImageWithRegistry(t *testing.T) {
	registry, repository, tag := SplitDockerImage("custom.registry/ubuntu")

	assert.Equal(t, "custom.registry", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "", tag)

	dockerImage := context.DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "custom.registry/ubuntu", dockerImage.String())
}

func TestSplitDockerImageWithRegistryAndTag(t *testing.T) {
	registry, repository, tag := SplitDockerImage("custom.registry/ubuntu:12.04")

	assert.Equal(t, "custom.registry", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := context.DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "custom.registry/ubuntu:12.04", dockerImage.String())
}

func TestSplitDockerImageWithRepositoryAndTag(t *testing.T) {
	registry, repository, tag := SplitDockerImage("ubuntu:12.04")

	assert.Equal(t, "", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := context.DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "ubuntu:12.04", dockerImage.String())
}

func TestSplitDockerImageWithPrivateRegistryPath(t *testing.T) {
	registry, repository, tag := SplitDockerImage("localhost:8888/ubuntu/foo:12.04")

	assert.Equal(t, "localhost:8888", registry)
	assert.Equal(t, "ubuntu/foo", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := context.DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "localhost:8888/ubuntu/foo:12.04", dockerImage.String())
}
func TestSplitDockerImageWithLocalRepositoryAndTag(t *testing.T) {
	registry, repository, tag := SplitDockerImage("localhost:8888/ubuntu:12.04")

	assert.Equal(t, "localhost:8888", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := context.DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "localhost:8888/ubuntu:12.04", dockerImage.String())
}

func TestParseHostUnix(t *testing.T) {
	proto, addr, err := parseHost("unix:///var/run/docker.sock")
	assert.NoError(t, err)
	assert.Equal(t, "unix", proto, "failed to parse unix:///var/run/docker.sock")
	assert.Equal(t, "/var/run/docker.sock", addr, "failed to parse unix:///var/run/docker.sock")
}

func TestParseHostUnixDefault(t *testing.T) {
	proto, addr, err := parseHost("")
	assert.NoError(t, err)
	assert.Equal(t, "unix", proto, "failed to parse ''")
	assert.Equal(t, "/var/run/docker.sock", addr, "failed to parse ''")
}

func TestParseHostUnixDefaultNoPath(t *testing.T) {
	proto, addr, err := parseHost("unix://")
	assert.NoError(t, err)
	assert.Equal(t, "unix", proto, "failed to parse unix://")
	assert.Equal(t, "/var/run/docker.sock", addr, "failed to parse unix://")
}

func TestParseHostTCP(t *testing.T) {
	proto, addr, err := parseHost("tcp://127.0.0.1:4243")
	assert.NoError(t, err)
	assert.Equal(t, "tcp", proto, "failed to parse tcp://127.0.0.1:4243")
	assert.Equal(t, "127.0.0.1:4243", addr, "failed to parse tcp://127.0.0.1:4243")
}

func TestParseHostTCPDefault(t *testing.T) {
	proto, addr, err := parseHost("tcp://:4243")
	assert.NoError(t, err)
	assert.Equal(t, "tcp", proto, "failed to parse tcp://:4243")
	assert.Equal(t, "127.0.0.1:4243", addr, "failed to parse tcp://:4243")
}

func TestParseHostSystemd(t *testing.T) {
	proto, addr, err := parseHost("fd://")
	assert.NoError(t, err)
	assert.Equal(t, "fd", proto, "failed to parse fd://")
	assert.Equal(t, "fd://", addr, "failed to parse fd://")
}

func assertParseHostError(t *testing.T, address string) {
	proto, addr, err := parseHost(address)
	message := fmt.Sprintf("should have failed to parse %v", address)
	assert.Error(t, err, message)
	assert.Equal(t, "", proto, message)
	assert.Equal(t, "", addr, message)
}

func TestParseHostTCPNoAddressError(t *testing.T) {
	assertParseHostError(t, "tcp://")
}

func TestParseHostTCPIncorrectBindAddressError(t *testing.T) {
	incorrectBindAdresses := []string{
		"tcp://127.0.0.1:4243:80",
		"tcp://127.0.0.1:",
		"tcp://127.0.0.1",
	}

	for _, address := range incorrectBindAdresses {
		assertParseHostError(t, address)
	}
}

func TestParseHostWrongProtocolError(t *testing.T) {
	assertParseHostError(t, "foo://")
}

func TestTlsEnabled(t *testing.T) {
	tls := tlsEnabled("foo", "bar", "baz")
	assert.False(t, tls)

	filepaths := map[string]string{
		"cert":   "",
		"caCert": "",
		"key":    "",
	}
	// Create temporary files
	for key := range filepaths {
		file, err := ioutil.TempFile("", key)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file.Name())
		filepaths[key] = file.Name()
	}

	tls = tlsEnabled(filepaths["cert"], filepaths["caCert"], filepaths["key"])
	assert.True(t, tls)
}
