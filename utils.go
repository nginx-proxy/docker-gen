package main

import (
	"errors"
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

	proto, host, err := parseHost(defaultEndpoint)
	if err != nil {
		return "", err
	}

	if proto == "unix" {
		exist, err := exists(host)
		if err != nil {
			return "", err
		}

		if !exist {
			return "", errors.New(host + " does not exist")
		}
	}

	return defaultEndpoint, nil
}
