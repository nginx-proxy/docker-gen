package integration

import (
	"testing"

	"github.com/nginx-proxy/docker-gen/integration/dockergen"
	"github.com/nginx-proxy/docker-gen/integration/internal/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
)

func TestDockergenContainers(t *testing.T) {
	images := []string{
		"nginxproxy/docker-gen:test-alpine",
		"nginxproxy/docker-gen:test-debian",
	}

	for _, image := range images {
		t.Run(image, func(t *testing.T) {
			runImage := image
			ctx := t.Context()

			label := "com.github.nginx-proxy.docker-gen"

			testOpts := []tc.ContainerCustomizer{
				tc.WithLabels(map[string]string{label: ""}),
				dockergen.WithContainerFilters("label=" + label),
				dockergen.WithCustomTemplate("test.tmpl"),
			}

			dockergenContainer, err := dockergen.Run(ctx, runImage, testOpts...)
			defer tc.CleanupContainer(t, dockergenContainer)
			require.NoError(t, err)

			var result helpers.TemplateResult
			err = dockergenContainer.UnmarshalRenderedTemplate(ctx, &result)
			require.NoError(t, err)

			assert.Equal(t, dockergenContainer.GetContainerID(), result.Docker.CurrentContainerID)
			assert.Equal(t, dockergenContainer.GetContainerID(), result.CurrentContainer.ID)
			assert.Len(t, result.Containers, 1)
			assert.Equal(t, dockergenContainer.GetContainerID(), result.Containers[0].ID)
		})
	}
}
