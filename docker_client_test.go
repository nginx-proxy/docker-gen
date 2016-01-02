package dockergen

import (
	"testing"
)

func TestSplitDockerImageRepository(t *testing.T) {
	registry, repository, tag := splitDockerImage("ubuntu")

	if registry != "" {
		t.Fail()
	}
	if repository != "ubuntu" {
		t.Fail()
	}
	if tag != "" {
		t.Fail()
	}

	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	if "ubuntu" != dockerImage.String() {
		t.Fail()
	}
}

func TestSplitDockerImageWithRegistry(t *testing.T) {
	registry, repository, tag := splitDockerImage("custom.registry/ubuntu")

	if registry != "custom.registry" {
		t.Fail()
	}
	if repository != "ubuntu" {
		t.Fail()
	}
	if tag != "" {
		t.Fail()
	}
	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	if "custom.registry/ubuntu" != dockerImage.String() {
		t.Fail()
	}

}

func TestSplitDockerImageWithRegistryAndTag(t *testing.T) {
	registry, repository, tag := splitDockerImage("custom.registry/ubuntu:12.04")

	if registry != "custom.registry" {
		t.Fail()
	}
	if repository != "ubuntu" {
		t.Fail()
	}
	if tag != "12.04" {
		t.Fail()
	}
	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	if "custom.registry/ubuntu:12.04" != dockerImage.String() {
		t.Fail()
	}

}

func TestSplitDockerImageWithRepositoryAndTag(t *testing.T) {
	registry, repository, tag := splitDockerImage("ubuntu:12.04")

	if registry != "" {
		t.Fail()
	}

	if repository != "ubuntu" {
		t.Fail()
	}

	if tag != "12.04" {
		t.Fail()
	}
	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	if "ubuntu:12.04" != dockerImage.String() {
		t.Fail()
	}
}

func TestSplitDockerImageWithPrivateRegistryPath(t *testing.T) {
	registry, repository, tag := splitDockerImage("localhost:8888/ubuntu/foo:12.04")

	if registry != "localhost:8888" {
		t.Fail()
	}

	if repository != "ubuntu/foo" {
		t.Fail()
	}

	if tag != "12.04" {
		t.Fail()
	}
	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	if "localhost:8888/ubuntu/foo:12.04" != dockerImage.String() {
		t.Fail()
	}
}
func TestSplitDockerImageWithLocalRepositoryAndTag(t *testing.T) {
	registry, repository, tag := splitDockerImage("localhost:8888/ubuntu:12.04")

	if registry != "localhost:8888" {
		t.Fatalf("registry does not match: expected %s got %s", "localhost:8888", registry)
	}

	if repository != "ubuntu" {
		t.Fatalf("repository does not match: expected %s got %s", "ubuntu", repository)
	}

	if tag != "12.04" {
		t.Fatalf("tag does not match: expected %s got %s", "12.04", tag)
	}
	dockerImage := DockerImage{
		Registry:   registry,
		Repository: repository,
		Tag:        tag,
	}
	if "localhost:8888/ubuntu:12.04" != dockerImage.String() {
		t.Fail()
	}

}

func TestParseHostUnix(t *testing.T) {
	proto, addr, err := parseHost("unix:///var/run/docker.sock")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if proto != "unix" || addr != "/var/run/docker.sock" {
		t.Fatal("failed to parse unix:///var/run/docker.sock")
	}
}

func TestParseHostUnixDefault(t *testing.T) {
	proto, addr, err := parseHost("")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if proto != "unix" || addr != "/var/run/docker.sock" {
		t.Fatal("failed to parse ''")
	}
}

func TestParseHostUnixDefaultNoPath(t *testing.T) {
	proto, addr, err := parseHost("unix://")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if proto != "unix" || addr != "/var/run/docker.sock" {
		t.Fatal("failed to parse unix://")
	}
}

func TestParseHostTCP(t *testing.T) {
	proto, addr, err := parseHost("tcp://127.0.0.1:4243")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if proto != "tcp" || addr != "127.0.0.1:4243" {
		t.Fatal("failed to parse tcp://127.0.0.1:4243")
	}
}

func TestParseHostTCPDefault(t *testing.T) {
	proto, addr, err := parseHost("tcp://:4243")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if proto != "tcp" || addr != "127.0.0.1:4243" {
		t.Fatal("failed to parse unix:///var/run/docker.sock")
	}
}
