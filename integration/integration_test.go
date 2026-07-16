package integration

import (
	"context"
	"testing"

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
			ctx := context.Background()

			dockergenContainer, err := startContainerWithTemplate(ctx, runImage, "test.tmpl")
			require.NoError(t, err)
			defer tc.CleanupContainer(t, dockergenContainer)

			var result templateResult
			err = dockergenContainer.unmarshalJsonFile(ctx, "/etc/docker-gen/rendered", &result)
			require.NoError(t, err)

			assert.Equal(t, dockergenContainer.GetContainerID(), result.Docker.CurrentContainerID)
			assert.Len(t, result.Containers, 1)
			assert.Equal(t, dockergenContainer.GetContainerID(), result.Containers[0].ID)
		})
	}
}
