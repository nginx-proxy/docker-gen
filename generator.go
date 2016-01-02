package dockergen

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type generator struct {
	Client                     *docker.Client
	Configs                    ConfigFile
	Endpoint                   string
	TLSVerify                  bool
	TLSCert, TLSCaCert, TLSKey string

	dockerInfo Server

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
	g.wg.Wait()

	return nil
}

func (g *generator) generateFromContainers(client *docker.Client) {
	containers, err := GetContainers(client)
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
					containers, err := GetContainers(client)
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
