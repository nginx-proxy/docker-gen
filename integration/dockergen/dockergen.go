package dockergen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/nginx-proxy/docker-gen/integration/internal/helpers"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// containerTemplatePath is the path (inside the container) to the template docker-gen will use.
	containerTemplatePath = "/etc/docker-gen/templates/test.tmpl"
	// containerRenderedPath is the path (inside the container) where the rendered template will be written by docker-gen.
	containerRenderedPath = "/etc/docker-gen/rendered"
)

// Container represents the docker-gen container type.
type Container struct {
	tc.Container
}

// WithCustomTemplate mounts a custom template file at /etc/docker-gen/templates/test.tmpl in the container.
//
// The templatePath must be an absolute or relative path to the template file on the host.
func WithCustomTemplate(templatePath string) tc.CustomizeRequestOption {
	return tc.WithFiles(tc.ContainerFile{
		HostFilePath:      templatePath,
		ContainerFilePath: containerTemplatePath,
		FileMode:          0o644,
	})
}

// WithContainerFilters sets the DOCKER_CONTAINER_FILTERS environment variable in the container to the given filters.
func WithContainerFilters(filters ...string) tc.CustomizeRequestOption {
	return tc.WithEnv(map[string]string{
		"DOCKER_CONTAINER_FILTERS": strings.Join(filters, ","),
	})
}

// WithDockerSocketBind mounts the Docker socket from the host into the container at /tmp/docker.sock (read-only).
//
// The dockerSocketPath must be an absolute path to the Docker socket on the host.
func WithDockerSocketBind(dockerSocketPath string) tc.CustomizeRequestOption {
	return tc.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
		hostConfig.Binds = []string{dockerSocketPath + ":/tmp/docker.sock:ro"}
	})
}

// Run creates an instance of the docker-gen container type.
func Run(ctx context.Context, img string, opts ...tc.ContainerCustomizer) (*Container, error) {
	dockerHostURL, err := helpers.GetDockerHostURL(ctx)
	if err != nil {
		return nil, err
	}

	if dockerHostURL.Scheme != "unix" {
		return nil, fmt.Errorf("unsupported Docker host scheme: %s", dockerHostURL.Scheme)
	}

	moduleOpts := make([]tc.ContainerCustomizer, 0, 5+len(opts))
	moduleOpts = append(moduleOpts,
		WithDockerSocketBind(dockerHostURL.Path),
		tc.WithCmd("-watch", containerTemplatePath, containerRenderedPath),
		tc.WithWaitStrategy(wait.ForFile(containerRenderedPath).WithStartupTimeout(time.Second*10)),
	)

	moduleOpts = append(moduleOpts, opts...)

	ctr, err := tc.Run(ctx, img, moduleOpts...)
	var c *Container
	if ctr != nil {
		c = &Container{Container: ctr}
	}

	if err != nil {
		return c, fmt.Errorf("run docker-gen: %w", err)
	}

	return c, err
}

// RenderedTemplate returns the content of the rendered template file from the container.
func (c *Container) RenderedTemplate(ctx context.Context) ([]byte, error) {
	output, err := c.CopyFileFromContainer(ctx, containerRenderedPath)
	if err != nil {
		return nil, fmt.Errorf("copy file from container: %w", err)
	}
	defer output.Close()

	content, err := io.ReadAll(output)
	if err != nil {
		return nil, fmt.Errorf("read file content: %w", err)
	}

	return content, nil
}

// UnmarshalRenderedTemplate unmarshals (as JSON) the content of the rendered template file from the container into the provided pointer.
func (c *Container) UnmarshalRenderedTemplate(ctx context.Context, pointer any) error {
	content, err := c.RenderedTemplate(ctx)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(content, pointer); err != nil {
		return fmt.Errorf("unmarshal rendered template: %w", err)
	}

	return nil
}
