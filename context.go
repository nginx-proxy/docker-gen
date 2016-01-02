package dockergen

import (
	"os"
	"sync"

	"github.com/fsouza/go-dockerclient"
)

var (
	mu         sync.RWMutex
	dockerInfo Docker
	dockerEnv  *docker.Env
)

type Context []*RuntimeContainer

func (c *Context) Env() map[string]string {
	return splitKeyValueSlice(os.Environ())
}

func (c *Context) Docker() Docker {
	mu.RLock()
	defer mu.RUnlock()
	return dockerInfo
}

func SetServerInfo(d *docker.Env) {
	mu.Lock()
	defer mu.Unlock()
	dockerInfo = Docker{
		Name:            d.Get("Name"),
		NumContainers:   d.GetInt("Containers"),
		NumImages:       d.GetInt("Images"),
		Version:         dockerEnv.Get("Version"),
		ApiVersion:      dockerEnv.Get("ApiVersion"),
		GoVersion:       dockerEnv.Get("GoVersion"),
		OperatingSystem: dockerEnv.Get("Os"),
		Architecture:    dockerEnv.Get("Arch"),
	}
}

func SetDockerEnv(d *docker.Env) {
	mu.Lock()
	defer mu.Unlock()
	dockerEnv = d
}

type Address struct {
	IP           string
	IP6LinkLocal string
	IP6Global    string
	Port         string
	HostPort     string
	Proto        string
	HostIP       string
}

type Network struct {
	IP                  string
	Name                string
	Gateway             string
	EndpointID          string
	IPv6Gateway         string
	GlobalIPv6Address   string
	MacAddress          string
	GlobalIPv6PrefixLen int
	IPPrefixLen         int
}

type Volume struct {
	Path      string
	HostPath  string
	ReadWrite bool
}

type RuntimeContainer struct {
	ID           string
	Addresses    []Address
	Networks     []Network
	Gateway      string
	Name         string
	Hostname     string
	Image        DockerImage
	Env          map[string]string
	Volumes      map[string]Volume
	Node         SwarmNode
	Labels       map[string]string
	IP           string
	IP6LinkLocal string
	IP6Global    string
	Mounts       []Mount
}

func (r *RuntimeContainer) Equals(o RuntimeContainer) bool {
	return r.ID == o.ID && r.Image == o.Image
}

func (r *RuntimeContainer) PublishedAddresses() []Address {
	mapped := []Address{}
	for _, address := range r.Addresses {
		if address.HostPort != "" {
			mapped = append(mapped, address)
		}
	}
	return mapped
}

type DockerImage struct {
	Registry   string
	Repository string
	Tag        string
}

func (i *DockerImage) String() string {
	ret := i.Repository
	if i.Registry != "" {
		ret = i.Registry + "/" + i.Repository
	}
	if i.Tag != "" {
		ret = ret + ":" + i.Tag
	}
	return ret
}

type SwarmNode struct {
	ID      string
	Name    string
	Address Address
}

type Mount struct {
	Name        string
	Source      string
	Destination string
	Driver      string
	Mode        string
	RW          bool
}

type Docker struct {
	Name            string
	NumContainers   int
	NumImages       int
	Version         string
	ApiVersion      string
	GoVersion       string
	OperatingSystem string
	Architecture    string
}
