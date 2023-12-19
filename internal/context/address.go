package context

import (
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

	return addresses
}
