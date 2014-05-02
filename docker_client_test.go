package main

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
