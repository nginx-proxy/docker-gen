package main

import "os"

func getEndpoint() string {
	defaultEndpoint := "unix:///var/run/docker.sock"
	if os.Getenv("DOCKER_HOST") != "" {
		defaultEndpoint = os.Getenv("DOCKER_HOST")
	}

	if endpoint != "" {
		defaultEndpoint = endpoint
	}

	return defaultEndpoint

}
