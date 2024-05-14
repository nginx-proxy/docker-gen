package generator

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/nginx-proxy/docker-gen/internal/config"
	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/nginx-proxy/docker-gen/internal/dockerclient"
	"github.com/nginx-proxy/docker-gen/internal/template"
	"github.com/nginx-proxy/docker-gen/internal/utils"
)

type generator struct {
	Client                     *docker.Client
	Configs                    config.ConfigFile
	Endpoint                   string
	TLSVerify                  bool
	TLSCert, TLSCaCert, TLSKey string
	All                        bool

	wg    sync.WaitGroup
	retry bool
}

type GeneratorConfig struct {
	Endpoint string

	TLSCert   string
	TLSKey    string
	TLSCACert string
	TLSVerify bool
	All       bool

	ConfigFile config.ConfigFile
}

func NewGenerator(gc GeneratorConfig) (*generator, error) {
	endpoint, err := dockerclient.GetEndpoint(gc.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("bad endpoint: %s", err)
	}

	client, err := dockerclient.NewDockerClient(endpoint, gc.TLSVerify, gc.TLSCert, gc.TLSCACert, gc.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %s", err)
	}

	apiVersion, err := client.Version()
	if err != nil {
		log.Printf("Error retrieving docker server version info: %s\n", err)
	}

	// Grab the docker daemon info once and hold onto it
	context.SetDockerEnv(apiVersion)

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

		sigChan, cleanup := newSignalChannel()
		defer cleanup()
		for {
			sig := <-sigChan
			log.Printf("Received signal: %s\n", sig)
			switch sig {
			case syscall.SIGHUP:
				g.generateFromContainers()
			case syscall.SIGTERM, syscall.SIGINT:
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
		changed := template.GenerateFile(config, containers)
		if !changed {
			log.Printf("Contents of %s did not change. Skipping notification '%s'", config.Dest, config.NotifyCmd)
			continue
		}
		g.runNotifyCmd(config)
		g.sendSignalToContainers(config)
		g.sendSignalToFilteredContainers(config)
	}
}

func (g *generator) generateAtInterval() {
	for _, cfg := range g.Configs.Config {

		if cfg.Interval == 0 {
			continue
		}

		log.Printf("Generating every %d seconds", cfg.Interval)
		g.wg.Add(1)
		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		go func(cfg config.Config) {
			defer g.wg.Done()

			sigChan, cleanup := newSignalChannel()
			defer cleanup()
			for {
				select {
				case <-ticker.C:
					containers, err := g.getContainers()
					if err != nil {
						log.Printf("Error listing containers: %s\n", err)
						continue
					}
					// ignore changed return value. always run notify command
					template.GenerateFile(cfg, containers)
					g.runNotifyCmd(cfg)
					g.sendSignalToContainers(cfg)
					g.sendSignalToFilteredContainers(cfg)
				case sig := <-sigChan:
					log.Printf("Received signal: %s\n", sig)
					switch sig {
					case syscall.SIGTERM, syscall.SIGINT:
						ticker.Stop()
						return
					}
				}
			}
		}(cfg)
	}
}

func (g *generator) generateFromEvents() {
	configs := g.Configs.FilterWatches()
	if len(configs.Config) == 0 {
		return
	}

	client := g.Client
	var watchers []chan *docker.APIEvents

	for _, cfg := range configs.Config {

		if !cfg.Watch {
			continue
		}

		g.wg.Add(1)
		watcher := make(chan *docker.APIEvents, 100)
		watchers = append(watchers, watcher)

		go func(cfg config.Config) {
			defer g.wg.Done()
			debouncedChan := newDebounceChannel(watcher, cfg.Wait)
			for range debouncedChan {
				containers, err := g.getContainers()
				if err != nil {
					log.Printf("Error listing containers: %s\n", err)
					continue
				}
				changed := template.GenerateFile(cfg, containers)
				if !changed {
					log.Printf("Contents of %s did not change. Skipping notification '%s'", cfg.Dest, cfg.NotifyCmd)
					continue
				}
				g.runNotifyCmd(cfg)
				g.sendSignalToContainers(cfg)
				g.sendSignalToFilteredContainers(cfg)
			}
		}(cfg)
	}

	// maintains docker client connection and passes events to watchers
	go func() {
		// channel will be closed by go-dockerclient
		eventChan := make(chan *docker.APIEvents, 100)
		sigChan, cleanup := newSignalChannel()
		defer cleanup()

		for {
			watching := false

			if client == nil {
				var err error
				endpoint, err := dockerclient.GetEndpoint(g.Endpoint)
				if err != nil {
					log.Printf("Bad endpoint: %s", err)
					time.Sleep(10 * time.Second)
					continue
				}
				client, err = dockerclient.NewDockerClient(endpoint, g.TLSVerify, g.TLSCert, g.TLSCaCert, g.TLSKey)
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

					watchedEvent := false

					switch event.Type {
					case "container":
						watchedContainerActions := []string{"start", "stop", "die", "health_status"}
						watchedEvent = slices.Contains(watchedContainerActions, event.Action)
					case "network":
						watchedNetworkActions := []string{"connect", "disconnect"}
						watchedEvent = slices.Contains(watchedNetworkActions, event.Action)
					}

					if watchedEvent {
						log.Printf("Received event %s for %s %s", event.Action, event.Type, event.Actor.ID[:12])
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
					case syscall.SIGTERM, syscall.SIGINT:
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

func (g *generator) runNotifyCmd(config config.Config) {
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

func (g *generator) sendSignalToContainer(container string, signal int) {
	log.Printf("Sending container '%s' signal '%v'", container, signal)

	if signal == -1 {
		if err := g.Client.RestartContainer(container, 10); err != nil {
			log.Printf("Error sending restarting container: %s", err)
		}
		return
	}

	killOpts := docker.KillContainerOptions{
		ID:     container,
		Signal: docker.Signal(signal),
	}
	if err := g.Client.KillContainer(killOpts); err != nil {
		log.Printf("Error sending signal to container: %s", err)
	}
}

func (g *generator) sendSignalToContainers(config config.Config) {
	if len(config.NotifyContainers) < 1 {
		return
	}

	for container, signal := range config.NotifyContainers {
		g.sendSignalToContainer(container, signal)
	}
}

func (g *generator) sendSignalToFilteredContainers(config config.Config) {
	if len(config.NotifyContainersFilter) < 1 {
		return
	}

	containers, err := g.Client.ListContainers(docker.ListContainersOptions{
		Filters: config.NotifyContainersFilter,
	})
	if err != nil {
		log.Printf("Error getting containers: %s", err)
		return
	}

	for _, container := range containers {
		g.sendSignalToContainer(container.ID, config.NotifyContainersSignal)
	}
}

func (g *generator) getContainers() ([]*context.RuntimeContainer, error) {
	apiInfo, err := g.Client.Info()
	if err != nil {
		log.Printf("Error retrieving docker server info: %s\n", err)
	} else {
		context.SetServerInfo(apiInfo)
	}

	apiContainers, err := g.Client.ListContainers(docker.ListContainersOptions{
		All:  g.All,
		Size: false,
	})
	if err != nil {
		return nil, err
	}

	apiNetworks, err := g.Client.ListNetworks()
	if err != nil {
		return nil, err
	}
	networks := make(map[string]docker.Network)
	for _, apiNetwork := range apiNetworks {
		networks[apiNetwork.Name] = apiNetwork
	}

	containers := []*context.RuntimeContainer{}
	for _, apiContainer := range apiContainers {
		opts := docker.InspectContainerOptions{ID: apiContainer.ID}
		container, err := g.Client.InspectContainerWithOptions(opts)
		if err != nil {
			log.Printf("Error inspecting container: %s: %s\n", apiContainer.ID, err)
			continue
		}

		registry, repository, tag := dockerclient.SplitDockerImage(container.Config.Image)
		runtimeContainer := &context.RuntimeContainer{
			ID:      container.ID,
			Created: container.Created,
			Image: context.DockerImage{
				Registry:   registry,
				Repository: repository,
				Tag:        tag,
			},
			State: context.State{
				Running: container.State.Running,
				Health: context.Health{
					Status: container.State.Health.Status,
				},
			},
			Name:         strings.TrimLeft(container.Name, "/"),
			Hostname:     container.Config.Hostname,
			Gateway:      container.NetworkSettings.Gateway,
			NetworkMode:  container.HostConfig.NetworkMode,
			Addresses:    []context.Address{},
			Networks:     []context.Network{},
			Env:          make(map[string]string),
			Volumes:      make(map[string]context.Volume),
			Node:         context.SwarmNode{},
			Labels:       make(map[string]string),
			IP:           container.NetworkSettings.IPAddress,
			IP6LinkLocal: container.NetworkSettings.LinkLocalIPv6Address,
			IP6Global:    container.NetworkSettings.GlobalIPv6Address,
		}

		adresses := context.GetContainerAddresses(container)
		runtimeContainer.Addresses = append(runtimeContainer.Addresses, adresses...)

		for k, v := range container.NetworkSettings.Networks {
			network := context.Network{
				IP:                  v.IPAddress,
				Name:                k,
				Gateway:             v.Gateway,
				EndpointID:          v.EndpointID,
				IPv6Gateway:         v.IPv6Gateway,
				GlobalIPv6Address:   v.GlobalIPv6Address,
				MacAddress:          v.MacAddress,
				GlobalIPv6PrefixLen: v.GlobalIPv6PrefixLen,
				IPPrefixLen:         v.IPPrefixLen,
				Internal:            networks[k].Internal,
			}

			runtimeContainer.Networks = append(runtimeContainer.Networks,
				network)
		}

		for k, v := range container.Volumes {
			runtimeContainer.Volumes[k] = context.Volume{
				Path:      k,
				HostPath:  v,
				ReadWrite: container.VolumesRW[k],
			}
		}
		if container.Node != nil {
			runtimeContainer.Node.ID = container.Node.ID
			runtimeContainer.Node.Name = container.Node.Name
			runtimeContainer.Node.Address = context.Address{
				IP: container.Node.IP,
			}
		}

		for _, v := range container.Mounts {
			runtimeContainer.Mounts = append(runtimeContainer.Mounts, context.Mount{
				Name:        v.Name,
				Source:      v.Source,
				Destination: v.Destination,
				Driver:      v.Driver,
				Mode:        v.Mode,
				RW:          v.RW,
			})
		}

		runtimeContainer.Env = utils.SplitKeyValueSlice(container.Config.Env)
		runtimeContainer.Labels = container.Config.Labels
		containers = append(containers, runtimeContainer)
	}
	return containers, nil

}

func newSignalChannel() (<-chan os.Signal, func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	return sig, func() { signal.Stop(sig) }
}

func newDebounceChannel(input chan *docker.APIEvents, wait *config.Wait) chan *docker.APIEvents {
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
