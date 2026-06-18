package context

import (
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

type FakePortBinding struct{}

var httpPort = docker.Port("80/tcp")
var httpPortBinding = docker.PortBinding{
	HostIP:   "100.100.100.100",
	HostPort: "8080",
}

var httpsPort = docker.Port("443/tcp")

var httpTestPort = docker.Port("8080/tcp")
var httpsTestPort = docker.Port("8443/tcp")

func TestGenerateContainerAddresses(t *testing.T) {
	testContainer := &docker.Container{
		Config: &docker.Config{
			ExposedPorts: map[docker.Port]struct{}{},
		},
		NetworkSettings: &docker.NetworkSettings{
			IPAddress:            "10.0.0.10",
			LinkLocalIPv6Address: "24",
			GlobalIPv6Address:    "10.0.0.1",
			Ports:                map[docker.Port][]docker.PortBinding{},
		},
	}
	testContainer.NetworkSettings.Ports[httpPort] = []docker.PortBinding{httpPortBinding}
	testContainer.NetworkSettings.Ports[httpsPort] = []docker.PortBinding{}

	addresses := GetContainerAddresses(testContainer)
	assert.Len(t, addresses, len(testContainer.NetworkSettings.Ports))
	assert.Contains(t, addresses, Address{
		IP:           "10.0.0.10",
		IP6LinkLocal: "24",
		IP6Global:    "10.0.0.1",
		Port:         "80",
		Proto:        "tcp",
		HostIP:       "100.100.100.100",
		HostPort:     "8080",
	})
	assert.Contains(t, addresses, Address{
		IP:           "10.0.0.10",
		IP6LinkLocal: "24",
		IP6Global:    "10.0.0.1",
		Port:         "443",
		Proto:        "tcp",
		HostIP:       "",
		HostPort:     "",
	})
}

func TestGenerateContainerAddressesWithExposedPorts(t *testing.T) {
	testContainer := &docker.Container{
		Config: &docker.Config{
			ExposedPorts: map[docker.Port]struct{}{},
		},
		NetworkSettings: &docker.NetworkSettings{
			IPAddress:            "10.0.0.10",
			LinkLocalIPv6Address: "24",
			GlobalIPv6Address:    "10.0.0.1",
			Ports:                map[docker.Port][]docker.PortBinding{},
		},
	}
	testContainer.NetworkSettings.Ports[httpPort] = []docker.PortBinding{}
	testContainer.NetworkSettings.Ports[httpsPort] = []docker.PortBinding{}
	testContainer.Config.ExposedPorts[httpPort] = struct{}{}
	testContainer.Config.ExposedPorts[httpsPort] = struct{}{}
	testContainer.Config.ExposedPorts[httpTestPort] = struct{}{}

	assert.Len(t, GetContainerAddresses(testContainer), 2)
}

func TestGenerateContainerAddressesWithNoPorts(t *testing.T) {
	testContainer := &docker.Container{
		Config: &docker.Config{
			ExposedPorts: map[docker.Port]struct{}{},
		},
		NetworkSettings: &docker.NetworkSettings{
			IPAddress:            "10.0.0.10",
			LinkLocalIPv6Address: "24",
			GlobalIPv6Address:    "10.0.0.1",
			Ports:                map[docker.Port][]docker.PortBinding{},
		},
	}
	testContainer.Config.ExposedPorts[httpTestPort] = FakePortBinding{}
	testContainer.Config.ExposedPorts[httpsTestPort] = FakePortBinding{}

	addresses := GetContainerAddresses(testContainer)
	assert.Len(t, addresses, len(testContainer.Config.ExposedPorts))
	assert.Contains(t, addresses, Address{
		IP:           "10.0.0.10",
		IP6LinkLocal: "24",
		IP6Global:    "10.0.0.1",
		Port:         "8080",
		Proto:        "tcp",
		HostIP:       "",
		HostPort:     "",
	})
	assert.Contains(t, addresses, Address{
		IP:           "10.0.0.10",
		IP6LinkLocal: "24",
		IP6Global:    "10.0.0.1",
		Port:         "8443",
		Proto:        "tcp",
		HostIP:       "",
		HostPort:     "",
	})
}

func TestSortAddresses(t *testing.T) {
	addresses := []Address{
		{IP: "10.0.0.10", Port: "8080", Proto: "tcp"},
		{IP: "10.0.0.10", Port: "80", Proto: "tcp"},
		{IP: "10.0.0.10", Port: "443", Proto: "tcp"},
		{IP: "10.0.0.10", Port: "53", Proto: "udp"},
		{IP: "10.0.0.10", Port: "53", Proto: "tcp"},
	}

	// Port sorts numerically (not lexically: "443" must not sort before "80"),
	// with Proto as the tie-breaker for equal ports (53/tcp before 53/udp).
	want := []Address{
		{IP: "10.0.0.10", Port: "53", Proto: "tcp"},
		{IP: "10.0.0.10", Port: "53", Proto: "udp"},
		{IP: "10.0.0.10", Port: "80", Proto: "tcp"},
		{IP: "10.0.0.10", Port: "443", Proto: "tcp"},
		{IP: "10.0.0.10", Port: "8080", Proto: "tcp"},
	}

	sortAddresses(addresses)
	assert.Equal(t, want, addresses)
}

func TestGetContainerAddressesSorted(t *testing.T) {
	testContainer := &docker.Container{
		Config: &docker.Config{
			ExposedPorts: map[docker.Port]struct{}{},
		},
		NetworkSettings: &docker.NetworkSettings{
			IPAddress: "10.0.0.10",
			Ports:     map[docker.Port][]docker.PortBinding{},
		},
	}
	// Insert ports out of numeric order; the map iteration order is random but
	// GetContainerAddresses must return them deterministically sorted by port.
	testContainer.NetworkSettings.Ports[httpTestPort] = []docker.PortBinding{} // 8080
	testContainer.NetworkSettings.Ports[httpPort] = []docker.PortBinding{}     // 80
	testContainer.NetworkSettings.Ports[httpsPort] = []docker.PortBinding{}    // 443

	addresses := GetContainerAddresses(testContainer)
	ports := make([]string, len(addresses))
	for i, a := range addresses {
		ports[i] = a.Port
	}
	assert.Equal(t, []string{"80", "443", "8080"}, ports)
}
