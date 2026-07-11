package context

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/nginx-proxy/docker-gen/internal/utils"
)

var (
	mu                   sync.RWMutex
	dockerInfo           Docker
	dockerEnv            *docker.Env
	hostnameRegex        = regexp.MustCompilePOSIX("^[[:alnum:]]{12}$")
	mountinfoPrefixRegex = regexp.MustCompilePOSIX("^[0-9]+ [0-9]+ [0-9]+:[0-9]+ /")
)

type Context []*RuntimeContainer

func (c *Context) Env() map[string]string {
	return utils.SplitKeyValueSlice(os.Environ())
}

func (c *Context) Docker() Docker {
	mu.RLock()
	defer mu.RUnlock()
	return dockerInfo
}

func SetServerInfo(d *docker.DockerInfo) {
	mu.Lock()
	defer mu.Unlock()
	dockerInfo = Docker{
		Name:               d.Name,
		NumContainers:      d.Containers,
		NumImages:          d.Images,
		Version:            dockerEnv.Get("Version"),
		ApiVersion:         dockerEnv.Get("ApiVersion"),
		GoVersion:          dockerEnv.Get("GoVersion"),
		OperatingSystem:    dockerEnv.Get("Os"),
		Architecture:       dockerEnv.Get("Arch"),
		CurrentContainerID: GetCurrentContainerID(),
	}
}

func SetDockerEnv(d *docker.Env) {
	mu.Lock()
	defer mu.Unlock()
	dockerEnv = d
}

type Network struct {
	IP                  string
	Name                string
	Aliases             []string
	Gateway             string
	EndpointID          string
	IPv6Gateway         string
	GlobalIPv6Address   string
	MacAddress          string
	GlobalIPv6PrefixLen int
	IPPrefixLen         int
	Internal            bool
}

type Volume struct {
	Path      string
	HostPath  string
	ReadWrite bool
}

type State struct {
	Running bool
	Health  Health
}

type Health struct {
	Status string
}

type RuntimeContainer struct {
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

func (r *RuntimeContainer) Equals(o RuntimeContainer) bool {
	return r.ID == o.ID && r.Image == o.Image
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

// GetCurrentContainerID attempts to extract the current container ID from the provided file paths.
// If no files paths are provided, it will default to /proc/1/cpuset, /proc/self/cgroup and /proc/self/mountinfo.
// It attempts to match the HOSTNAME first then use the fallback method, and returns with the first valid match.
func GetCurrentContainerID(filepaths ...string) (id string) {
	if len(filepaths) == 0 {
		filepaths = []string{"/proc/1/cpuset", "/proc/self/cgroup", "/proc/self/mountinfo"}
	}

	// We try to match a 64 character hex string starting with the hostname first
	for _, filepath := range filepaths {
		if id = readContainerIDFromFile(filepath, matchContainerIDWithHostname); len(id) == 64 {
			return
		}
	}

	// If we didn't get any ID that matches the hostname, fall back to matching the first 64 character hex string
	for _, filepath := range filepaths {
		if id = readContainerIDFromFile(filepath, matchContainerID); len(id) == 64 {
			return
		}
	}

	return
}

// readContainerIDFromFile reads a file and attempts to match a container ID.
// idMatcher is a function that extracts a container ID from a string.
func readContainerIDFromFile(filepath string, idMatcher func(string) string) (id string) {
	file, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle long /proc/mountinfo lines
	// 1MB is usually sufficient for even the largest mountinfo entries
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		strLines := scanner.Text()
		if id = idMatcher(strLines); len(id) == 64 {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

func matchContainerID(lines string) string {
	return matchContainerIDWithRegex("([[:alnum:]]{64})", lines)
}

func matchContainerIDWithHostname(lines string) string {
	hostname := os.Getenv("HOSTNAME")

	if hostnameRegex.MatchString(hostname) {
		regex := fmt.Sprintf("(%s[[:alnum:]]{52})", hostname)

		return matchContainerIDWithRegex(regex, lines)
	}
	return ""
}

func matchContainerIDWithRegex(regex, lines string) (containerID string) {
	matchAndExtract := func(pattern string) string {
		re := regexp.MustCompilePOSIX(pattern)
		submatches := re.FindStringSubmatch(lines)
		if len(submatches) >= 2 {
			return submatches[1]
		}
		return ""
	}

	// Attempt to detect if we're on a line from a /proc/<pid>/mountinfo file and modify the regexp accordingly
	// https://www.kernel.org/doc/Documentation/filesystems/proc.txt section 3.5
	if mountinfoPrefixRegex.MatchString(lines) {
		// Match on containers/<id> while also supporting file-anchored paths and Podman paths that insert a userdata segment before those files.
		// See https://github.com/nginx-proxy/docker-gen/issues/452 and https://github.com/nginx-proxy/nginx-proxy/issues/2759
		if containerID = matchAndExtract(fmt.Sprintf("containers/%v", regex)); len(containerID) == 64 {
			return
		}
		if containerID = matchAndExtract(fmt.Sprintf("/%v/(userdata/)?(hostname|hosts|resolv\\.conf)", regex)); len(containerID) == 64 {
			return
		}
		return ""
	}

	return matchAndExtract(regex)
}
