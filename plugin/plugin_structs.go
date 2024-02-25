package plugin

//go:generate $GOPATH/bin/easyjson -all $GOFILE

import "time"

type (
	PluginContext struct {
		Containers []*RuntimeContainer
		Env        map[string]string
		Docker     Docker
	}

	Network struct {
		IP                  string
		Name                string
		Gateway             string
		EndpointID          string
		IPv6Gateway         string
		GlobalIPv6Address   string
		MacAddress          string
		GlobalIPv6PrefixLen int
		IPPrefixLen         int
		Internal            bool
	}

	Volume struct {
		Path      string
		HostPath  string
		ReadWrite bool
	}

	State struct {
		Running bool
		Health  Health
	}

	Health struct {
		Status string
	}

	Address struct {
		IP           string
		IP6LinkLocal string
		IP6Global    string
		Port         string
		HostPort     string
		Proto        string
		HostIP       string
	}

	RuntimeContainer struct {
		ID           string
		Created      time.Time
		Addresses    []Address
		Networks     []Network
		Gateway      string
		Name         string
		Hostname     string
		NetworkMode  string
		Image        DockerImage
		Env          map[string]string
		Volumes      map[string]Volume
		Node         SwarmNode
		Labels       map[string]string
		IP           string
		IP6LinkLocal string
		IP6Global    string
		Mounts       []Mount
		State        State
	}

	DockerImage struct {
		Registry   string
		Repository string
		Tag        string
	}

	SwarmNode struct {
		ID      string
		Name    string
		Address Address
	}

	Mount struct {
		Name        string
		Source      string
		Destination string
		Driver      string
		Mode        string
		RW          bool
	}

	Docker struct {
		Name               string
		NumContainers      int
		NumImages          int
		Version            string
		ApiVersion         string
		GoVersion          string
		OperatingSystem    string
		Architecture       string
		CurrentContainerID string
	}
)
