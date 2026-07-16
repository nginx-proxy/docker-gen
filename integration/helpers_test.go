package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/moby/moby/api/types/container"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type templateResult struct {
	Docker     docker             `json:"Docker"`
	Env        map[string]string  `json:"Env"`
	Containers []runtimeContainer `json:"Containers"`
}

type docker struct {
	CurrentContainerID string
}

type runtimeContainer struct {
	ID   string
	Name string
}

type dockergenContainer struct {
	tc.Container
}

func (c *dockergenContainer) unmarshalJsonFile(ctx context.Context, path string, pointer any) (err error) {
	output, err := c.CopyFileFromContainer(ctx, path)
	if err != nil {
		return
	}
	defer output.Close()

	content, err := io.ReadAll(output)
	if err != nil {
		return
	}

	err = json.Unmarshal(content, pointer)
	if err != nil {
		return
	}

	return
}

func startContainerWithTemplate(ctx context.Context, image string, path string, opts ...tc.ContainerCustomizer) (*dockergenContainer, error) {
	templateFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := templateFile.Close(); err != nil {
			log.Printf("Failed to close template file: %v", err)
		}
	}()

	templateFileMount := tc.WithFiles(tc.ContainerFile{
		Reader:            templateFile,
		ContainerFilePath: "/etc/docker-gen/templates/test.tmpl",
		FileMode:          0o644,
	})

	opts = append(opts, templateFileMount)

	return startContainerWithOpts(ctx, image, opts...)
}

func getDockerHostURL(ctx context.Context) (*url.URL, error) {
	client, err := tc.NewDockerClientWithOpts(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	url, err := url.Parse(client.DaemonHost())
	if err != nil {
		return nil, err
	}

	return url, nil
}

func startContainerWithOpts(ctx context.Context, image string, opts ...tc.ContainerCustomizer) (*dockergenContainer, error) {
	dockerHostURL, err := getDockerHostURL(ctx)
	if err != nil {
		return nil, err
	}

	if dockerHostURL.Scheme != "unix" {
		return nil, fmt.Errorf("unsupported Docker host scheme: %s", dockerHostURL.Scheme)
	}

	runOpts := []tc.ContainerCustomizer{
		tc.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Binds = []string{dockerHostURL.Path + ":/tmp/docker.sock:ro"}
		}),
		tc.WithLabels(map[string]string{
			"com.github.nginx-proxy.docker-gen": "",
		}),
		tc.WithCmd(
			"-watch",
			"-container-filter",
			"label=com.github.nginx-proxy.docker-gen",
			"/etc/docker-gen/templates/test.tmpl",
			"/etc/docker-gen/rendered",
		),
		tc.WithWaitStrategy(wait.ForFile("/etc/docker-gen/rendered").WithStartupTimeout(time.Second * 10)),
	}

	runOpts = slices.Concat(
		runOpts,
		opts,
	)

	ctr, err := tc.Run(
		ctx,
		image,
		runOpts...,
	)

	var dockergenCtr *dockergenContainer
	if ctr != nil {
		dockergenCtr = &dockergenContainer{Container: ctr}
	}

	return dockergenCtr, err
}
