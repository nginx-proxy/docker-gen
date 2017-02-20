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

	"github.com/docker/docker/api/types/swarm"
	"github.com/fsouza/go-dockerclient"
	"strconv"
)

type generator struct {
	Client                     *docker.Client
	Configs                    ConfigFile
	Endpoint                   string
	TLSVerify                  bool
	TLSCert, TLSCaCert, TLSKey string
	All                        bool

	wg    sync.WaitGroup
	retry bool
	Swarm bool
}

type GeneratorConfig struct {
	Endpoint string

	TLSCert   string
	TLSKey    string
	TLSCACert string
	TLSVerify bool
	All       bool

	Swarm      bool
	ConfigFile ConfigFile
}

type SwarmInfo struct {
	Name string
	Body string
	Time int64
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
		log.Printf("Error retrieving docker server version info: %s\n", err)
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
		All:       gc.All,
		Configs:   gc.ConfigFile,
		retry:     true,
		Swarm:     gc.Swarm,
	}, nil
}

func (g *generator) Generate() error {
	g.generateFromContainers()
	g.generateAtInterval()
	g.generateFromEvents()
	g.generateFromSignals()
	g.wg.Wait()

	return nil
}

func (g *generator) generateFromSignals() {
	var hasWatcher bool
	for _, config := range g.Configs.Config {
		if config.Watch {
			hasWatcher = true
			break
		}
	}

	// If none of the configs need to watch for events, don't watch for signals either
	if !hasWatcher {
		return
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		sigChan := newSignalChannel()
		for {
			sig := <-sigChan
			log.Printf("Received signal: %s\n", sig)
			switch sig {
			case syscall.SIGHUP:
				g.generateFromContainers()
			case syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT:
				// exit when context is done
				return
			}
		}
	}()
}

func (g *generator) generateFromContainers() {
	containers, err := g.getContainers()
	if err != nil {
		log.Printf("Error listing containers: %s\n", err)
		return
	}
	for _, config := range g.Configs.Config {
		changed := GenerateFile(config, containers)
		if !changed {
			log.Printf("Contents of %s did not change. Skipping notification '%s'", config.Dest, config.NotifyCmd)
			continue
		}
		g.runNotifyCmd(config)
		g.sendSignalToContainer(config)
	}
}

func (g *generator) generateAtInterval() {
	for _, config := range g.Configs.Config {

		if config.Interval == 0 {
			continue
		}

		log.Printf("Generating every %d seconds", config.Interval)
		g.wg.Add(1)
		ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
		go func(config Config) {
			defer g.wg.Done()

			sigChan := newSignalChannel()
			for {
				select {
				case <-ticker.C:
					containers, err := g.getContainers()
					if err != nil {
						log.Printf("Error listing containers: %s\n", err)
						continue
					}
					// ignore changed return value. always run notify command
					GenerateFile(config, containers)
					g.runNotifyCmd(config)
					g.sendSignalToContainer(config)
				case sig := <-sigChan:
					log.Printf("Received signal: %s\n", sig)
					switch sig {
					case syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT:
						ticker.Stop()
						return
					}
				}
			}
		}(config)
	}
}

func (g *generator) generateFromEvents() {
	configs := g.Configs.FilterWatches()
	if len(configs.Config) == 0 {
		return
	}

	client := g.Client
	var watchers []chan *docker.APIEvents

	for _, config := range configs.Config {

		if !config.Watch {
			continue
		}

		g.wg.Add(1)

		go func(config Config, watcher chan *docker.APIEvents) {
			defer g.wg.Done()
			watchers = append(watchers, watcher)

			debouncedChan := newDebounceChannel(watcher, config.Wait)
			for _ = range debouncedChan {
				containers, err := g.getContainers()
				if err != nil {
					log.Printf("Error listing containers: %s\n", err)
					continue
				}
				changed := GenerateFile(config, containers)
				if !changed {
					log.Printf("Contents of %s did not change. Skipping notification '%s'", config.Dest, config.NotifyCmd)
					continue
				}
				g.runNotifyCmd(config)
				g.sendSignalToContainer(config)
			}
		}(config, make(chan *docker.APIEvents, 100))
	}

	// maintains docker client connection and passes events to watchers
	go func() {
		// channel will be closed by go-dockerclient
		eventChan := make(chan *docker.APIEvents, 100)
		sigChan := newSignalChannel()

		for {
			watching := false

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
			}

			for {
				if client == nil {
					break
				}
				if !watching {
					err := client.AddEventListener(eventChan)
					if err != nil && err != docker.ErrListenerAlreadyExists {
						log.Printf("Error registering docker event listener: %s", err)
						time.Sleep(10 * time.Second)
						continue
					}
					watching = true
					log.Println("Watching docker events")
					// sync all configs after resuming listener
					g.generateFromContainers()
				}
				select {
				case event, ok := <-eventChan:
					if !ok {
						log.Printf("Docker daemon connection interrupted")
						if watching {
							client.RemoveEventListener(eventChan)
							watching = false
							client = nil
						}
						if !g.retry {
							// close all watchers and exit
							for _, watcher := range watchers {
								close(watcher)
							}
							return
						}
						// recreate channel and attempt to resume
						eventChan = make(chan *docker.APIEvents, 100)
						time.Sleep(10 * time.Second)
						break
					}
					if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
						log.Printf("Received event %s for container %s", event.Status, event.ID[:12])
						// fanout event to all watchers
						for _, watcher := range watchers {
							watcher <- event
						}
					}
				case <-time.After(10 * time.Second):
					// check for docker liveness
					err := client.Ping()
					if err != nil {
						log.Printf("Unable to ping docker daemon: %s", err)
						if watching {
							client.RemoveEventListener(eventChan)
							watching = false
							client = nil
						}
					}
				case sig := <-sigChan:
					log.Printf("Received signal: %s\n", sig)
					switch sig {
					case syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT:
						// close all watchers and exit
						for _, watcher := range watchers {
							close(watcher)
						}
						return
					}
				}
			}
		}
	}()
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

func (g *generator) sendSignalToContainer(config Config) {
	if len(config.NotifyContainers) < 1 {
		return
	}

	for container, signal := range config.NotifyContainers {
		log.Printf("Sending container '%s' signal '%v'", container, signal)
		killOpts := docker.KillContainerOptions{
			ID:     container,
			Signal: signal,
		}
		if err := g.Client.KillContainer(killOpts); err != nil {
			log.Printf("Error sending signal to container: %s", err)
		}
	}
}

type Cluster struct {
	ID string
}

type Swarm struct {
	Cluster Cluster
}

func (g *generator) getContainers() ([]*RuntimeContainer, error) {
	apiInfo, err := g.Client.Info()
	if err != nil {
		log.Printf("Error retrieving docker server info: %s\n", err)
	} else {
		SetServerInfo(apiInfo)
	}
	if g.Swarm && apiInfo.Swarm.Cluster.ID != "" {
		return g.getContainersFromSwarm()
	} else {
		return g.getContainersFromLocalDocker()
	}

}

type ImagesCache struct {
	cache map[string]*docker.Image
}

func (e TextError) Error() string { return e.msg }

type TextError struct {
	msg string
}

func NewImagesCache(client *docker.Client) *ImagesCache {
	ret := &ImagesCache{
		cache: make(map[string]*docker.Image),
	}
	images, error := client.ListImages(docker.ListImagesOptions{
		Digests: true,
	})
	if error != nil {
		log.Println("Error ListImages .")
	} else {
		for _, img := range images {
			inspectedImage, err := client.InspectImage(img.ID)
			if err == nil {
				for _, dig := range inspectedImage.RepoDigests {
					ID := dig[strings.Index(dig, "@")+1:]
					ret.cache[ID] = inspectedImage
				}
			}
		}
	}
	return ret
}

func (ic *ImagesCache) getImage(client *docker.Client, imageDigest string) (*docker.Image, error) {
	ID := imageDigest[strings.Index(imageDigest, "@")+1:]
	i := ic.cache[ID]
	if i == nil {
		return nil, TextError{"Unable to find image of given digest"}
	} else {
		return i, nil
	}

}

func (g *generator) getContainersFromSwarm() ([]*RuntimeContainer, error) {
	tasks, err := g.Client.ListTasks(docker.ListTasksOptions{})
	imageCache := NewImagesCache(g.Client)
	if err != nil {
		return nil, err
	}
	containers := []*RuntimeContainer{}
	for _, task := range tasks {
		container, err := g.Client.InspectTask(task.ID)
		if err != nil {
			log.Printf("Error inspecting task: %s: %s\n", task.ID, err)
			continue
		}
		service, err := g.Client.InspectService(container.ServiceID)
		if err != nil {
			log.Printf("Error inspecting service: %s: %s\n", container.ServiceID, err)
			continue
		}
		node, err := g.Client.InspectNode(container.NodeID)
		if err != nil {
			log.Printf("Error inspecting Node: %s: %s\n", container.NodeID, err)
			continue
		}
		registry, repository, tag := splitDockerImage(container.Spec.ContainerSpec.Image)
		runtimeContainer := &RuntimeContainer{
			ID: container.Status.ContainerStatus.ContainerID,
			Image: DockerImage{
				Registry:   registry,
				Repository: repository,
				Tag:        tag,
			},
			State: State{
				Running: container.Status.State == swarm.TaskStateRunning,
			},
			Name:      strings.TrimLeft(container.Name, "/"),
			Hostname:  container.Spec.ContainerSpec.Hostname,
			Addresses: []Address{},
			Networks:  []Network{},
			Env:       make(map[string]string),
			Volumes:   make(map[string]Volume),
			Node: SwarmNode{
				ID:   container.NodeID,
				Name: node.Spec.Name,
				Address: Address{
					HostIP: node.Status.Addr,
					IP:     node.Status.Addr,
				},
			},
			Labels:       make(map[string]string),
			Gateway:      "",
			IP:           "",
			IP6LinkLocal: "",
			IP6Global:    "",
		}
		for _, net := range container.NetworksAttachments {
			ingressNet := net.Network.Spec.Name == "ingres"
			for _, addr := range net.Addresses {
				addressNoNet := addr[:strings.Index(addr, "/")]
				for _, port := range service.Spec.EndpointSpec.Ports {
					var exposedPort string
					if ingressNet {
						exposedPort = fmt.Sprint(port.PublishedPort)
					} else {
						exposedPort = fmt.Sprint(port.TargetPort)
					}
					address := Address{
						IP:       addressNoNet,
						Port:     exposedPort,
						HostPort: fmt.Sprint(port.TargetPort),
						Proto:    string(port.Protocol),
					}
					runtimeContainer.Addresses = append(runtimeContainer.Addresses,
						address)
				}
			}

			ip := ""
			netLen := 0
			if len(net.Addresses) > 0 {
				ip = net.Addresses[0]
				netLen, _ = strconv.Atoi(ip[strings.Index(ip, "/")+1:])
				ip = ip[:strings.Index(ip, "/")]
			}
			var config swarm.IPAMConfig
			if len(net.Network.IPAMOptions.Configs) > 0 {
				config = net.Network.IPAMOptions.Configs[0]
			}

			network := Network{
				IP:                  ip,
				Name:                net.Network.Spec.Name,
				Gateway:             config.Gateway,
				EndpointID:          "",
				IPv6Gateway:         "",
				GlobalIPv6Address:   "",
				MacAddress:          "",
				GlobalIPv6PrefixLen: 0,
				IPPrefixLen:         netLen,
			}

			runtimeContainer.Networks = append(runtimeContainer.Networks,
				network)
		}
		if len(runtimeContainer.Addresses) == 0 {
			//we havent found any ports so we look them in image
			image, err := imageCache.getImage(g.Client, container.Spec.ContainerSpec.Image)
			if err == nil && len(image.Config.ExposedPorts) > 0 {
				for _, net := range container.NetworksAttachments {
					ingressNet := net.Network.Spec.Name == "ingres"
					if ingressNet {
						continue
					}

					for _, addr := range net.Addresses {
						addressNoNet := addr[:strings.Index(addr, "/")]
						for k := range image.Config.ExposedPorts {
							address := Address{
								IP:       addressNoNet,
								Port:     k.Port(),
								HostPort: k.Port(),
								Proto:    k.Proto(),
							}
							runtimeContainer.Addresses = append(runtimeContainer.Addresses,
								address)
						}
					}
				}
			}

		}
		for _, v := range container.Spec.ContainerSpec.Mounts {
			mode := ""
			if v.TmpfsOptions != nil {
				mode = v.TmpfsOptions.Mode.String()
			}
			driver := ""
			if v.VolumeOptions != nil && v.VolumeOptions.DriverConfig != nil {
				driver = v.VolumeOptions.DriverConfig.Name
			}
			runtimeContainer.Mounts = append(runtimeContainer.Mounts, Mount{
				Name:        "",
				Source:      v.Source,
				Destination: v.Target,
				Driver:      driver,
				Mode:        mode,
				RW:          !v.ReadOnly,
			})
		}

		runtimeContainer.Env = splitKeyValueSlice(container.Spec.ContainerSpec.Env)
		runtimeContainer.Labels = container.Spec.ContainerSpec.Labels
		containers = append(containers, runtimeContainer)
	}
	return containers, nil
}

func (g *generator) getContainersFromLocalDocker() ([]*RuntimeContainer, error) {
	apiContainers, err := g.Client.ListContainers(docker.ListContainersOptions{
		All:  g.All,
		Size: false,
	})
	if err != nil {
		return nil, err
	}

	containers := []*RuntimeContainer{}
	for _, apiContainer := range apiContainers {
		container, err := g.Client.InspectContainer(apiContainer.ID)
		if err != nil {
			log.Printf("Error inspecting container: %s: %s\n", apiContainer.ID, err)
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
			State: State{
				Running: container.State.Running,
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

func newSignalChannel() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	return sig
}

func newDebounceChannel(input chan *docker.APIEvents, wait *Wait) chan *docker.APIEvents {
	if wait == nil {
		return input
	}
	if wait.Min == 0 {
		return input
	}

	output := make(chan *docker.APIEvents, 100)

	go func() {
		var (
			event    *docker.APIEvents
			minTimer <-chan time.Time
			maxTimer <-chan time.Time
		)

		defer close(output)

		for {
			select {
			case buffer, ok := <-input:
				if !ok {
					return
				}
				event = buffer
				minTimer = time.After(wait.Min)
				if maxTimer == nil {
					maxTimer = time.After(wait.Max)
				}
			case <-minTimer:
				log.Println("Debounce minTimer fired")
				minTimer, maxTimer = nil, nil
				output <- event
			case <-maxTimer:
				log.Println("Debounce maxTimer fired")
				minTimer, maxTimer = nil, nil
				output <- event
			}
		}
	}()

	return output
}
