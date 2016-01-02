package dockergen

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type generator struct {
	Client                     *docker.Client
	Configs                    ConfigFile
	Endpoint                   string
	TLSVerify                  bool
	TLSCert, TLSCaCert, TLSKey string

	wg sync.WaitGroup
}

type GeneratorConfig struct {
	Endpoint string

	TLSCert   string
	TLSKey    string
	TLSCACert string
	TLSVerify bool

	ConfigFile ConfigFile
}

func NewGenerator(gc GeneratorConfig) (*generator, error) {
	endpoint, err := GetEndpoint(gc.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Bad endpoint: %s", err)
	}

	client, err := NewDockerClient(endpoint, gc.TLSVerify, gc.TLSCert, gc.TLSCACert, gc.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("Unable to create docker client: %s", err)
	}

	apiVersion, err := client.Version()
	if err != nil {
		log.Printf("error retrieving docker server version info: %s\n", err)
	}

	// Grab the docker daemon info once and hold onto it
	SetDockerEnv(apiVersion)

	return &generator{
		Client:    client,
		Endpoint:  gc.Endpoint,
		TLSVerify: gc.TLSVerify,
		TLSCert:   gc.TLSCert,
		TLSCaCert: gc.TLSCACert,
		TLSKey:    gc.TLSKey,
		Configs:   gc.ConfigFile,
	}, nil
}

func (g *generator) Generate() error {
	g.generateFromContainers(g.Client)
	g.generateAtInterval(g.Client, g.Configs)
	g.generateFromEvents(g.Client, g.Configs)
	g.generateFromSignals()
	g.wg.Wait()

	return nil
}

func (g *generator) generateFromSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		for {
			sig := <-sigs
			log.Printf("Received signal: %s\n", sig)
			switch sig {
			case syscall.SIGHUP:
				g.generateFromContainers(g.Client)
			case syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT:
				// exit when context is done
				return
			}
		}
	}()
}

func (g *generator) generateFromContainers(client *docker.Client) {
	containers, err := g.getContainers(client)
	if err != nil {
		log.Printf("error listing containers: %s\n", err)
		return
	}
	for _, config := range g.Configs.Config {
		changed := GenerateFile(config, containers)
		if !changed {
			log.Printf("Contents of %s did not change. Skipping notification '%s'", config.Dest, config.NotifyCmd)
			continue
		}
		g.runNotifyCmd(config)
		g.sendSignalToContainer(client, config)
	}
}

func (g *generator) generateAtInterval(client *docker.Client, configs ConfigFile) {
	for _, config := range configs.Config {

		if config.Interval == 0 {
			continue
		}

		log.Printf("Generating every %d seconds", config.Interval)
		g.wg.Add(1)
		ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
		quit := make(chan struct{})
		configCopy := config
		go func() {
			defer g.wg.Done()
			for {
				select {
				case <-ticker.C:
					containers, err := g.getContainers(client)
					if err != nil {
						log.Printf("Error listing containers: %s\n", err)
						continue
					}
					// ignore changed return value. always run notify command
					GenerateFile(configCopy, containers)
					g.runNotifyCmd(configCopy)
					g.sendSignalToContainer(client, configCopy)
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
}

func (g *generator) generateFromEvents(client *docker.Client, configs ConfigFile) {
	configs = configs.FilterWatches()
	if len(configs.Config) == 0 {
		return
	}

	g.wg.Add(1)
	defer g.wg.Done()

	for {
		if client == nil {
			var err error
			endpoint, err := GetEndpoint(g.Endpoint)
			if err != nil {
				log.Printf("Bad endpoint: %s", err)
				time.Sleep(10 * time.Second)
				continue
			}

			client, err = NewDockerClient(endpoint, g.TLSVerify, g.TLSCert, g.TLSCaCert, g.TLSKey)
			if err != nil {
				log.Printf("Unable to connect to docker daemon: %s", err)
				time.Sleep(10 * time.Second)
				continue
			}
			g.generateFromContainers(client)
		}

		eventChan := make(chan *docker.APIEvents, 100)
		defer close(eventChan)

		watching := false
		for {

			if client == nil {
				break
			}
			err := client.Ping()
			if err != nil {
				log.Printf("Unable to ping docker daemon: %s", err)
				if watching {
					client.RemoveEventListener(eventChan)
					watching = false
					client = nil
				}
				time.Sleep(10 * time.Second)
				break

			}

			if !watching {
				err = client.AddEventListener(eventChan)
				if err != nil && err != docker.ErrListenerAlreadyExists {
					log.Printf("Error registering docker event listener: %s", err)
					time.Sleep(10 * time.Second)
					continue
				}
				watching = true
				log.Println("Watching docker events")
			}

			select {

			case event := <-eventChan:
				if event == nil {
					if watching {
						client.RemoveEventListener(eventChan)
						watching = false
						client = nil
					}
					break
				}

				if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
					log.Printf("Received event %s for container %s", event.Status, event.ID[:12])
					g.generateFromContainers(client)
				}
			case <-time.After(10 * time.Second):
				// check for docker liveness
			}

		}
	}
}

func (g *generator) runNotifyCmd(config Config) {
	if config.NotifyCmd == "" {
		return
	}

	log.Printf("Running '%s'", config.NotifyCmd)
	cmd := exec.Command("/bin/sh", "-c", config.NotifyCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running notify command: %s, %s\n", config.NotifyCmd, err)
	}
	if config.NotifyOutput {
		for _, line := range strings.Split(string(out), "\n") {
			if line != "" {
				log.Printf("[%s]: %s", config.NotifyCmd, line)
			}
		}
	}
}

func (g *generator) sendSignalToContainer(client *docker.Client, config Config) {
	if len(config.NotifyContainers) < 1 {
		return
	}

	for container, signal := range config.NotifyContainers {
		log.Printf("Sending container '%s' signal '%v'", container, signal)
		killOpts := docker.KillContainerOptions{
			ID:     container,
			Signal: signal,
		}
		if err := client.KillContainer(killOpts); err != nil {
			log.Printf("Error sending signal to container: %s", err)
		}
	}
}

func (g *generator) getContainers(client *docker.Client) ([]*RuntimeContainer, error) {
	apiInfo, err := client.Info()
	if err != nil {
		log.Printf("error retrieving docker server info: %s\n", err)
	}

	SetServerInfo(apiInfo)

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

		for _, v := range container.Mounts {
			runtimeContainer.Mounts = append(runtimeContainer.Mounts, Mount{
				Name:        v.Name,
				Source:      v.Source,
				Destination: v.Destination,
				Driver:      v.Driver,
				Mode:        v.Mode,
				RW:          v.RW,
			})
		}

		runtimeContainer.Env = splitKeyValueSlice(container.Config.Env)
		runtimeContainer.Labels = container.Config.Labels
		containers = append(containers, runtimeContainer)
	}
	return containers, nil

}
