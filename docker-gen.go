package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/fsouza/go-dockerclient"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"os/signal"

	"strings"
	"syscall"
)

var (
	watch       bool
	notifyCmd   string
	onlyExposed bool
	configFile  string
	configs     ConfigFile
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
	Image     string
	Env       map[string]string
}

type Config struct {
	Template    string
	Dest        string
	Watch       bool
	NotifyCmd   string
	OnlyExposed bool
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
	println("Usage: docker-gen [-config file] [-watch=false] [-notify=\"restart xyz\"] <template> [<dest>]")
}

func newConn() (*httputil.ClientConn, error) {
	conn, err := net.Dial("unix", "/var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	return httputil.NewClientConn(conn, nil), nil
}

func getEvents() chan *Event {
	eventChan := make(chan *Event, 100)
	go func() {
		defer close(eventChan)

	restart:

		c, err := newConn()
		if err != nil {
			fmt.Printf("cannot connect to docker: %s\n", err)
			return
		}
		defer c.Close()

		req, err := http.NewRequest("GET", "/events", nil)
		if err != nil {
			fmt.Printf("bad request for events: %s\n", err)
			return
		}

		resp, err := c.Do(req)
		if err != nil {
			fmt.Printf("cannot connect to events endpoint: %s\n", err)
			return
		}
		defer resp.Body.Close()

		// handle signals to stop the socket
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		go func() {
			for sig := range sigChan {
				fmt.Printf("received signal '%v', exiting\n", sig)

				c.Close()
				close(eventChan)
				os.Exit(0)
			}
		}()

		dec := json.NewDecoder(resp.Body)
		for {
			var event *Event
			if err := dec.Decode(&event); err != nil {
				if err == io.EOF {
					break
				}
				fmt.Printf("cannot decode json: %s\n", err)
				goto restart
			}
			eventChan <- event
		}
		fmt.Printf("closing event channel\n")
	}()
	return eventChan
}

func generateFromContainers(client *docker.Client) {
	apiContainers, err := client.ListContainers(docker.ListContainersOptions{
		All: false,
	})
	if err != nil {
		fmt.Printf("error listing containers: %s\n", err)
		return
	}

	containers := []*RuntimeContainer{}
	for _, apiContainer := range apiContainers {
		container, err := client.InspectContainer(apiContainer.ID)
		if err != nil {
			fmt.Printf("error inspecting container: %s: %s\n", apiContainer.ID, err)
			continue
		}

		runtimeContainer := &RuntimeContainer{
			ID:        container.ID,
			Image:     container.Config.Image,
			Addresses: []Address{},
			Env:       make(map[string]string),
		}
		for k, _ := range container.NetworkSettings.Ports {
			runtimeContainer.Addresses = append(runtimeContainer.Addresses,
				Address{
					IP:   container.NetworkSettings.IPAddress,
					Port: k.Port(),
				})
		}

		for _, entry := range container.Config.Env {
			parts := strings.Split(entry, "=")
			runtimeContainer.Env[parts[0]] = parts[1]
		}

		containers = append(containers, runtimeContainer)
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
		fmt.Printf("error running notify command: %s, %s\n", config.NotifyCmd, err)
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

func main() {
	flag.BoolVar(&watch, "watch", false, "watch for container changes")
	flag.BoolVar(&onlyExposed, "only-exposed", false, "only include containers with exposed ports")
	flag.StringVar(&notifyCmd, "notify", "", "run command after template is regenerated")
	flag.StringVar(&configFile, "config", "", "config file with template directives")
	flag.Parse()

	if flag.NArg() < 1 && configFile == "" {
		usage()
		os.Exit(1)
	}

	if configFile != "" {
		err := loadConfig(configFile)
		if err != nil {
			fmt.Printf("error loading config %s: %s\n", configFile, err)
			os.Exit(1)
		}
	} else {
		config := Config{
			Template:    flag.Arg(0),
			Dest:        flag.Arg(1),
			Watch:       watch,
			NotifyCmd:   notifyCmd,
			OnlyExposed: onlyExposed,
		}
		configs = ConfigFile{
			Config: []Config{config}}
	}

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)

	if err != nil {
		panic(err)
	}

	generateFromContainers(client)

	configs = configs.filterWatches()

	if len(configs.Config) == 0 {
		return
	}

	eventChan := getEvents()
	for {
		event := <-eventChan
		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			generateFromContainers(client)
		}
	}

}
