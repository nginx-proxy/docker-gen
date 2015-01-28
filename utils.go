package main

import (
	"os"
)

func getEndpoint() (string, error) {
	defaultEndpoint := "unix:///var/run/docker.sock"
	if os.Getenv("DOCKER_HOST") != "" {
		defaultEndpoint = os.Getenv("DOCKER_HOST")
	}

	if endpoint != "" {
		defaultEndpoint = endpoint
	}

	_, _, err := parseHost(defaultEndpoint)
	if err != nil {
		return "", err
	}

	return defaultEndpoint, nil
}
