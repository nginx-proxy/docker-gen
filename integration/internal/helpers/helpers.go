package helpers

import (
	"context"
	"net/url"

	tc "github.com/testcontainers/testcontainers-go"
)

type TemplateResult struct {
	Docker           docker             `json:"Docker"`
	Env              map[string]string  `json:"Env"`
	Containers       []runtimeContainer `json:"Containers"`
	CurrentContainer runtimeContainer   `json:"CurrentContainer"`
}

type docker struct {
	CurrentContainerID string
}

type runtimeContainer struct {
	ID   string
	Name string
}

// GetDockerHostURL returns the Docker host URL for the current testcontainers environment.
func GetDockerHostURL(ctx context.Context) (*url.URL, error) {
	client, err := tc.NewDockerClientWithOpts(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	dockerHostURL, err := url.Parse(client.DaemonHost())
	if err != nil {
		return nil, err
	}

	return dockerHostURL, nil
}
