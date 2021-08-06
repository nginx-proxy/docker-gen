package dockergen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitDockerImageRepository(t *testing.T) {
	registry, repository, tag := splitDockerImage("ubuntu")

	assert.Equal(t, "", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "", tag)

	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "ubuntu", dockerImage.String())
}

func TestSplitDockerImageWithRegistry(t *testing.T) {
	registry, repository, tag := splitDockerImage("custom.registry/ubuntu")

	assert.Equal(t, "custom.registry", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "", tag)

	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "custom.registry/ubuntu", dockerImage.String())
}

func TestSplitDockerImageWithRegistryAndTag(t *testing.T) {
	registry, repository, tag := splitDockerImage("custom.registry/ubuntu:12.04")

	assert.Equal(t, "custom.registry", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "custom.registry/ubuntu:12.04", dockerImage.String())
}

func TestSplitDockerImageWithRepositoryAndTag(t *testing.T) {
	registry, repository, tag := splitDockerImage("ubuntu:12.04")

	assert.Equal(t, "", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "ubuntu:12.04", dockerImage.String())
}

func TestSplitDockerImageWithPrivateRegistryPath(t *testing.T) {
	registry, repository, tag := splitDockerImage("localhost:8888/ubuntu/foo:12.04")

	assert.Equal(t, "localhost:8888", registry)
	assert.Equal(t, "ubuntu/foo", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	assert.Equal(t, "localhost:8888/ubuntu/foo:12.04", dockerImage.String())
}
func TestSplitDockerImageWithLocalRepositoryAndTag(t *testing.T) {
	registry, repository, tag := splitDockerImage("localhost:8888/ubuntu:12.04")

	assert.Equal(t, "localhost:8888", registry)
	assert.Equal(t, "ubuntu", repository)
	assert.Equal(t, "12.04", tag)

	dockerImage := DockerImage{
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
