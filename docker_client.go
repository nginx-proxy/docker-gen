package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

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

func newConn() (*httputil.ClientConn, error) {
	endpoint := getEndpoint()

	proto, addr, err := parseHost(endpoint)
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial(proto, addr)
	if err != nil {
		return nil, err
	}

	return httputil.NewClientConn(conn, nil), nil
}

func getEvents() chan *Event {
	eventChan := make(chan *Event, 100)
	go func() {
		defer close(eventChan)

		for {

			c, err := newConn()
			if err != nil {
				log.Printf("cannot connect to docker: %s\n", err)
				time.Sleep(10 * time.Second)
				continue
			}

			req, err := http.NewRequest("GET", "/events", nil)
			if err != nil {
				log.Printf("bad request for events: %s\n", err)
				c.Close()
				time.Sleep(10 * time.Second)
				continue
			}

			resp, err := c.Do(req)
			if err != nil {
				log.Printf("cannot connect to events endpoint: %s\n", err)
				c.Close()
				time.Sleep(10 * time.Second)
				continue
			}

			// handle signals to stop the socket
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
			go func() {
				for sig := range sigChan {
					log.Printf("received signal '%v', exiting\n", sig)

					c.Close()
					resp.Body.Close()
					close(eventChan)
					os.Exit(0)
				}
			}()

			dec := json.NewDecoder(resp.Body)
			for {
				var event *Event
				if err := dec.Decode(&event); err != nil || event.Status == "" {
					if err == io.EOF || (event != nil && event.Status == "") {
						log.Printf("connection closed")
						break
					}
					log.Printf("cannot decode json: %s\n", err)
					c.Close()
					resp.Body.Close()
					break
				}

				eventChan <- event
			}
		}
	}()
	return eventChan
}

func getContainers(client *docker.Client) ([]*RuntimeContainer, error) {

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
			Name:      strings.TrimLeft(container.Name, "/"),
			Gateway:   container.NetworkSettings.Gateway,
			Addresses: []Address{},
			Env:       make(map[string]string),
		}
		for k, v := range container.NetworkSettings.Ports {
			address := Address{
				IP:   container.NetworkSettings.IPAddress,
				Port: k.Port(),
			}
			if len(v) > 0 {
				address.HostPort = v[0].HostPort
			}
			runtimeContainer.Addresses = append(runtimeContainer.Addresses,
				address)

		}

		for _, entry := range container.Config.Env {
			parts := strings.Split(entry, "=")
			runtimeContainer.Env[parts[0]] = parts[1]
		}

		containers = append(containers, runtimeContainer)
	}
	return containers, nil

}
