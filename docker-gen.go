package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	buildVersion            string
	version                 bool
	watch                   bool
	notifyCmd               string
	notifySigHUPContainerID string
	onlyExposed             bool
	onlyPublished           bool
	configFile              string
	configs                 ConfigFile
	interval                int
	endpoint                string
	tlsCert                 string
	tlsKey                  string
	tlsCaCert               string
	tlsVerify               bool
	wg                      sync.WaitGroup
)

type Event struct {
	ContainerID string `json:"id"`
	Status      string `json:"status"`
	Image       string `json:"from"`
}

type Address struct {
	IP       string
	Port     string
	HostPort string
	Proto    string
}

type RuntimeContainer struct {
	ID        string
	Addresses []Address
	Gateway   string
	Name      string
	Image     DockerImage
	Env       map[string]string
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

type Config struct {
	Template         string
	Dest             string
	Watch            bool
	NotifyCmd        string
	NotifyContainers map[string]docker.Signal
	OnlyExposed      bool
	OnlyPublished    bool
	Interval         int
}

type ConfigFile struct {
	Config []Config
}

type Context []*RuntimeContainer

func (c *Context) Env() map[string]string {

	env := make(map[string]string)
	for _, i := range os.Environ() {
		parts := strings.Split(i, "=")
		env[parts[0]] = parts[1]
	}
	return env
}

func (c *ConfigFile) filterWatches() ConfigFile {
	configWithWatches := []Config{}

	for _, config := range c.Config {
		if config.Watch {
			configWithWatches = append(configWithWatches, config)
		}
	}
	return ConfigFile{
		Config: configWithWatches,
	}
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

func usage() {
	println("Usage: docker-gen [-config file] [-watch=false] [-notify=\"restart xyz\"] [-notify-sighup=\"container-ID\"] [-interval=0] [-endpoint tcp|unix://..] [-tlscert file] [-tlskey file] [-tlscacert file] [-tlsverify] <template> [<dest>]")
}

func generateFromContainers(client *docker.Client) {
	containers, err := getContainers(client)
	if err != nil {
		log.Printf("error listing containers: %s\n", err)
		return
	}
	for _, config := range configs.Config {
		changed := generateFile(config, containers)
		if !changed {
			log.Printf("Contents of %s did not change. Skipping notification '%s'", config.Dest, config.NotifyCmd)
			continue
		}
		runNotifyCmd(config)
		sendSignalToContainer(client, config)
	}
}

func runNotifyCmd(config Config) {
	if config.NotifyCmd == "" {
		return
	}

	log.Printf("Running '%s'", config.NotifyCmd)
	cmd := exec.Command("/bin/sh", "-c", config.NotifyCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("error running notify command: %s, %s\n", config.NotifyCmd, err)
		log.Print(string(out))
	}
}

func sendSignalToContainer(client *docker.Client, config Config) {
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

func loadConfig(file string) error {
	_, err := toml.DecodeFile(file, &configs)
	if err != nil {
		return err
	}
	return nil
}

func generateAtInterval(client *docker.Client, configs ConfigFile) {
	for _, config := range configs.Config {

		if config.Interval == 0 {
			continue
		}

		log.Printf("Generating every %d seconds", config.Interval)
		wg.Add(1)
		ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
		quit := make(chan struct{})
		configCopy := config
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ticker.C:
					containers, err := getContainers(client)
					if err != nil {
						log.Printf("error listing containers: %s\n", err)
						continue
					}
					// ignore changed return value. always run notify command
					generateFile(configCopy, containers)
					runNotifyCmd(configCopy)
					sendSignalToContainer(client, configCopy)
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
}

func generateFromEvents(client *docker.Client, configs ConfigFile) {
	configs = configs.filterWatches()
	if len(configs.Config) == 0 {
		return
	}

	wg.Add(1)
	defer wg.Done()

	eventChan := make(chan *docker.APIEvents, 100)
	defer close(eventChan)

	err := client.AddEventListener((chan<- *docker.APIEvents)(eventChan))
	if err != nil {
		log.Fatalf("Unable to add docker event listener: %s", err)
	}
	defer client.RemoveEventListener(eventChan)

	log.Println("Watching docker events")
	for {
		event := <-eventChan

		if event == nil {
			continue
		}

		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			log.Printf("Received event %s for container %s", event.Status, event.ID[:12])
			generateFromContainers(client)
		}
	}
}

func initFlags() {
	flag.BoolVar(&version, "version", false, "show version")
	flag.BoolVar(&watch, "watch", false, "watch for container changes")
	flag.BoolVar(&onlyExposed, "only-exposed", false, "only include containers with exposed ports")
	flag.BoolVar(&onlyPublished, "only-published", false, "only include containers with published ports (implies -only-exposed)")
	flag.StringVar(&notifyCmd, "notify", "", "run command after template is regenerated")
	flag.StringVar(&notifySigHUPContainerID, "notify-sighup", "", "send HUP signal to container.  Equivalent to `docker kill -s HUP container-ID`")
	flag.StringVar(&configFile, "config", "", "config file with template directives")
	flag.IntVar(&interval, "interval", 0, "notify command interval (s)")
	flag.StringVar(&endpoint, "endpoint", "", "docker api endpoint")
	flag.StringVar(&tlsCert, "tlscert", "", "path to TLS client certificate file")
	flag.StringVar(&tlsKey, "tlskey", "", "path to TLS client key file")
	flag.StringVar(&tlsCaCert, "tlscacert", "", "path to TLS CA certificate file")
	flag.BoolVar(&tlsVerify, "tlsverify", false, "verify docker daemon's TLS certicate")
	flag.Parse()
}

func main() {
	initFlags()

	if version {
		fmt.Println(buildVersion)
		return
	}

	if flag.NArg() < 1 && configFile == "" {
		usage()
		os.Exit(1)
	}

	if configFile != "" {
		err := loadConfig(configFile)
		if err != nil {
			log.Fatalf("error loading config %s: %s\n", configFile, err)
		}
	} else {
		config := Config{
			Template:         flag.Arg(0),
			Dest:             flag.Arg(1),
			Watch:            watch,
			NotifyCmd:        notifyCmd,
			NotifyContainers: make(map[string]docker.Signal),
			OnlyExposed:      onlyExposed,
			OnlyPublished:    onlyPublished,
			Interval:         interval,
		}
		if notifySigHUPContainerID != "" {
			config.NotifyContainers[notifySigHUPContainerID] = docker.SIGHUP
		}
		configs = ConfigFile{
			Config: []Config{config}}
	}

	endpoint, err := getEndpoint()
	if err != nil {
		log.Fatalf("Bad endpoint: %s", err)
	}

	var client *docker.Client
	if strings.HasPrefix(endpoint, "unix:") {
		client, err = docker.NewClient(endpoint)
	} else if tlsVerify || tlsCert != "" || tlsKey != "" || tlsCaCert != "" {
		if tlsVerify {
			if tlsCaCert == "" {
				log.Fatal("TLS verification was requested, but no -tlscacert was provided")
			}
		}

		client, err = docker.NewTLSClient(endpoint, tlsCert, tlsKey, tlsCaCert)
	} else {
		client, err = docker.NewClient(endpoint)
	}
	if err != nil {
		log.Fatalf("Unable to create docker client: %s", err)
	}

	generateFromContainers(client)
	generateAtInterval(client, configs)
	generateFromEvents(client, configs)
	wg.Wait()
}
