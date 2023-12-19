package dockerclient

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/nginx-proxy/docker-gen/internal/utils"
)

func GetEndpoint(endpoint string) (string, error) {
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

func NewDockerClient(endpoint string, tlsVerify bool, tlsCert, tlsCaCert, tlsKey string) (*docker.Client, error) {
	if strings.HasPrefix(endpoint, "unix:") {
		return docker.NewClient(endpoint)
	} else if tlsVerify || tlsEnabled(tlsCert, tlsCaCert, tlsKey) {
		if tlsVerify {
			if e, err := utils.PathExists(tlsCaCert); !e || err != nil {
				return nil, errors.New("TLS verification was requested, but CA cert does not exist")
			}
		}

		return docker.NewTLSClient(endpoint, tlsCert, tlsKey, tlsCaCert)
	}
	return docker.NewClient(endpoint)
}

func tlsEnabled(tlsCert, tlsCaCert, tlsKey string) bool {
	for _, v := range []string{tlsCert, tlsCaCert, tlsKey} {
		if e, err := utils.PathExists(v); e && err == nil {
			return true
		}
	}
	return false
}

type DockerContainer struct {
}

// based off of https://github.com/dotcloud/docker/blob/2a711d16e05b69328f2636f88f8eac035477f7e4/utils/utils.go
func parseHost(addr string) (string, string, error) {

	var (
		proto string
		host  string
		port  int
	)
	addr = strings.TrimSpace(addr)
	switch {
	case addr == "tcp://":
		return "", "", fmt.Errorf("invalid bind address format: %s", addr)
	case strings.HasPrefix(addr, "unix://"):
		proto = "unix"
		addr = strings.TrimPrefix(addr, "unix://")
		if addr == "" {
			addr = "/var/run/docker.sock"
		}
	case strings.HasPrefix(addr, "tcp://"):
		proto = "tcp"
		addr = strings.TrimPrefix(addr, "tcp://")
	case strings.HasPrefix(addr, "fd://"):
		return "fd", addr, nil
	case addr == "":
		proto = "unix"
		addr = "/var/run/docker.sock"
	default:
		if strings.Contains(addr, "://") {
			return "", "", fmt.Errorf("invalid bind address protocol: %s", addr)
		}
		proto = "tcp"
	}

	if proto != "unix" && strings.Contains(addr, ":") {
		hostParts := strings.Split(addr, ":")
		if len(hostParts) != 2 {
			return "", "", fmt.Errorf("invalid bind address format: %s", addr)
		}
		if hostParts[0] != "" {
			host = hostParts[0]
		} else {
			host = "127.0.0.1"
		}

		if p, err := strconv.Atoi(hostParts[1]); err == nil && p != 0 {
			port = p
		} else {
			return "", "", fmt.Errorf("invalid bind address format: %s", addr)
		}

	} else if proto == "tcp" && !strings.Contains(addr, ":") {
		return "", "", fmt.Errorf("invalid bind address format: %s", addr)
	} else {
		host = addr
	}
	if proto == "unix" {
		return proto, host, nil

	}
	return proto, fmt.Sprintf("%s:%d", host, port), nil
}

func SplitDockerImage(img string) (string, string, string) {
	index := 0
	repository := img
	var registry, tag string
	if strings.Contains(img, "/") {
		separator := strings.Index(img, "/")
		registry = img[index:separator]
		index = separator + 1
		repository = img[index:]
	}

	if strings.Contains(repository, ":") {
		separator := strings.Index(repository, ":")
		tag = repository[separator+1:]
		repository = repository[0:separator]
	}

	return registry, repository, tag
}
