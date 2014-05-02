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
	watch       bool
	notifyCmd   string
	onlyExposed bool
	configFile  string
	configs     ConfigFile
	interval    int
	endpoint    string
	wg          sync.WaitGroup
)

type Event struct {
	ContainerId string `json:"id"`
	Status      string `json:"status"`
	Image       string `json:"from"`
}

type Address struct {
	IP   string
	Port string
}
type RuntimeContainer struct {
	ID        string
	Addresses []Address
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
	Template    string
	Dest        string
	Watch       bool
	NotifyCmd   string
	OnlyExposed bool
	Interval    int
}

type ConfigFile struct {
	Config []Config
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

func usage() {
	println("Usage: docker-gen [-config file] [-watch=false] [-notify=\"restart xyz\"] [-interval=0] [-endpoint tcp|unix://..] <template> [<dest>]")
}

func generateFromContainers(client *docker.Client) {
	containers, err := getContainers(client)
	if err != nil {
		log.Printf("error listing containers: %s\n", err)
		return
	}
	for _, config := range configs.Config {
		changed := generateFile(config, containers)
		if changed {
			runNotifyCmd(config)
		}
	}
}

func runNotifyCmd(config Config) {
	if config.NotifyCmd == "" {
		return
	}

	args := strings.Split(config.NotifyCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("error running notify command: %s, %s\n", config.NotifyCmd, err)
		fmt.Println(string(out))
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
	eventChan := getEvents()
	for {
		event := <-eventChan
		if event == nil {
			return
		}
		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			generateFromContainers(client)
		}
	}
}

func main() {
	flag.BoolVar(&watch, "watch", false, "watch for container changes")
	flag.BoolVar(&onlyExposed, "only-exposed", false, "only include containers with exposed ports")
	flag.StringVar(&notifyCmd, "notify", "", "run command after template is regenerated")
	flag.StringVar(&configFile, "config", "", "config file with template directives")
	flag.IntVar(&interval, "interval", 0, "notify command interval (s)")
	flag.StringVar(&endpoint, "endpoint", "", "docker api endpoint")
	flag.Parse()

	if flag.NArg() < 1 && configFile == "" {
		usage()
		os.Exit(1)
	}

	if configFile != "" {
		err := loadConfig(configFile)
		if err != nil {
			log.Printf("error loading config %s: %s\n", configFile, err)
			os.Exit(1)
		}
	} else {
		config := Config{
			Template:    flag.Arg(0),
			Dest:        flag.Arg(1),
			Watch:       watch,
			NotifyCmd:   notifyCmd,
			OnlyExposed: onlyExposed,
			Interval:    interval,
		}
		configs = ConfigFile{
			Config: []Config{config}}
	}

	if endpoint == "" && os.Getenv("DOCKER_HOST") != "" {
		endpoint = os.Getenv("DOCKER_HOST")
	} else {
		endpoint = "unix:///var/run/docker.sock"
	}

	client, err := docker.NewClient(endpoint)

	if err != nil {
		log.Fatalf("Unable to parse %s: %s", endpoint, err)
	}

	generateFromContainers(client)
	generateAtInterval(client, configs)
	generateFromEvents(client, configs)
	wg.Wait()
}
