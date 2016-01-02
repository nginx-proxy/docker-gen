package dockergen

import "os"

type Context []*RuntimeContainer

func (c *Context) Env() map[string]string {
	return splitKeyValueSlice(os.Environ())
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
	Server       Server
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

type Server struct {
	Name          string
	NumContainers int
	NumImages     int
	Docker        Docker
}

type Docker struct {
	Version         string
	ApiVersion      string
	GoVersion       string
	OperatingSystem string
	Architecture    string
}
