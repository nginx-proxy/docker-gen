package context

import (
	"sort"
	"strconv"

	docker "github.com/fsouza/go-dockerclient"
)

type Address struct {
	IP           string
	IP6LinkLocal string
	IP6Global    string
	Port         string
	HostPort     string
	Proto        string
	HostIP       string
}

func renderAddress(container *docker.Container, port docker.Port) Address {
	return Address{
		IP:           container.NetworkSettings.IPAddress,
		IP6LinkLocal: container.NetworkSettings.LinkLocalIPv6Address,
		IP6Global:    container.NetworkSettings.GlobalIPv6Address,
		Port:         port.Port(),
		Proto:        port.Proto(),
	}
}

func GetContainerAddresses(container *docker.Container) []Address {
	addresses := []Address{}

	for port, bindings := range container.NetworkSettings.Ports {
		address := renderAddress(container, port)

		if len(bindings) > 0 {
			address.HostPort = bindings[0].HostPort
			address.HostIP = bindings[0].HostIP
		}

		addresses = append(addresses, address)
	}

	if len(addresses) == 0 {
		// internal docker network has empty 'container.NetworkSettings.Ports'
		for port := range container.Config.ExposedPorts {
			address := renderAddress(container, port)
			addresses = append(addresses, address)
		}
	}

	sortAddresses(addresses)

	return addresses
}

// sortAddresses sorts addresses in place by port (numeric), then proto, host port, host IP and IP.
func sortAddresses(addresses []Address) {
	sort.Slice(addresses, func(i, j int) bool {
		a, b := addresses[i], addresses[j]
		pa, _ := strconv.Atoi(a.Port)
		pb, _ := strconv.Atoi(b.Port)
		if pa != pb {
			return pa < pb
		}
		if a.Proto != b.Proto {
			return a.Proto < b.Proto
		}
		if a.HostPort != b.HostPort {
			return a.HostPort < b.HostPort
		}
		if a.HostIP != b.HostIP {
			return a.HostIP < b.HostIP
		}
		return a.IP < b.IP
	})
}
