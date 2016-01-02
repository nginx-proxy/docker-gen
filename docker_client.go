package dockergen

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

func NewDockerClient(endpoint string, tlsVerify bool, tlsCert, tlsCaCert, tlsKey string) (*docker.Client, error) {
	if strings.HasPrefix(endpoint, "unix:") {
		return docker.NewClient(endpoint)
	} else if tlsVerify || tlsEnabled(tlsCert, tlsCaCert, tlsKey) {
		if tlsVerify {
			if e, err := pathExists(tlsCaCert); !e || err != nil {
				return nil, errors.New("TLS verification was requested, but CA cert does not exist")
			}
		}

		return docker.NewTLSClient(endpoint, tlsCert, tlsKey, tlsCaCert)
	}
	return docker.NewClient(endpoint)
}

func tlsEnabled(tlsCert, tlsCaCert, tlsKey string) bool {
	for _, v := range []string{tlsCert, tlsCaCert, tlsKey} {
		if e, err := pathExists(v); e && err == nil {
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
		return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
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
			return "", "", fmt.Errorf("Invalid bind address protocol: %s", addr)
		}
		proto = "tcp"
	}

	if proto != "unix" && strings.Contains(addr, ":") {
		hostParts := strings.Split(addr, ":")
		if len(hostParts) != 2 {
			return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
		}
		if hostParts[0] != "" {
			host = hostParts[0]
		} else {
			host = "127.0.0.1"
		}

		if p, err := strconv.Atoi(hostParts[1]); err == nil && p != 0 {
			port = p
		} else {
			return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
		}

	} else if proto == "tcp" && !strings.Contains(addr, ":") {
		return "", "", fmt.Errorf("Invalid bind address format: %s", addr)
	} else {
		host = addr
	}
	if proto == "unix" {
		return proto, host, nil

	}
	return proto, fmt.Sprintf("%s:%d", host, port), nil
}

func splitDockerImage(img string) (string, string, string) {
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

func GetContainers(client *docker.Client) ([]*RuntimeContainer, error) {
	apiContainers, err := client.ListContainers(docker.ListContainersOptions{
		All:  false,
		Size: false,
	})
	if err != nil {
		return nil, err
	}

	containers := []*RuntimeContainer{}
	for _, apiContainer := range apiContainers {
		container, err := client.InspectContainer(apiContainer.ID)
		if err != nil {
			log.Printf("error inspecting container: %s: %s\n", apiContainer.ID, err)
			continue
		}

		registry, repository, tag := splitDockerImage(container.Config.Image)
		runtimeContainer := &RuntimeContainer{
			ID: container.ID,
			Image: DockerImage{
				Registry:   registry,
				Repository: repository,
				Tag:        tag,
			},
			Name:         strings.TrimLeft(container.Name, "/"),
			Hostname:     container.Config.Hostname,
			Gateway:      container.NetworkSettings.Gateway,
			Addresses:    []Address{},
			Networks:     []Network{},
			Env:          make(map[string]string),
			Volumes:      make(map[string]Volume),
			Node:         SwarmNode{},
			Labels:       make(map[string]string),
			IP:           container.NetworkSettings.IPAddress,
			IP6LinkLocal: container.NetworkSettings.LinkLocalIPv6Address,
			IP6Global:    container.NetworkSettings.GlobalIPv6Address,
		}
		for k, v := range container.NetworkSettings.Ports {
			address := Address{
				IP:           container.NetworkSettings.IPAddress,
				IP6LinkLocal: container.NetworkSettings.LinkLocalIPv6Address,
				IP6Global:    container.NetworkSettings.GlobalIPv6Address,
				Port:         k.Port(),
				Proto:        k.Proto(),
			}
			if len(v) > 0 {
				address.HostPort = v[0].HostPort
				address.HostIP = v[0].HostIP
			}
			runtimeContainer.Addresses = append(runtimeContainer.Addresses,
				address)

		}
		for k, v := range container.NetworkSettings.Networks {
			network := Network{
				IP:                  v.IPAddress,
				Name:                k,
				Gateway:             v.Gateway,
				EndpointID:          v.EndpointID,
				IPv6Gateway:         v.IPv6Gateway,
				GlobalIPv6Address:   v.GlobalIPv6Address,
				MacAddress:          v.MacAddress,
				GlobalIPv6PrefixLen: v.GlobalIPv6PrefixLen,
				IPPrefixLen:         v.IPPrefixLen,
			}

			runtimeContainer.Networks = append(runtimeContainer.Networks,
				network)
		}
		for k, v := range container.Volumes {
			runtimeContainer.Volumes[k] = Volume{
				Path:      k,
				HostPath:  v,
				ReadWrite: container.VolumesRW[k],
			}
		}
		if container.Node != nil {
			runtimeContainer.Node.ID = container.Node.ID
			runtimeContainer.Node.Name = container.Node.Name
			runtimeContainer.Node.Address = Address{
				IP: container.Node.IP,
			}
		}

		runtimeContainer.Env = splitKeyValueSlice(container.Config.Env)
		runtimeContainer.Labels = container.Config.Labels
		containers = append(containers, runtimeContainer)
	}
	return containers, nil

}

// pathExists returns whether the given file or directory exists or not
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
