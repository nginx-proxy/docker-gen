package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"text/template"
)

var (
	watch       bool
	notifyCmd   string
	onlyExposed bool
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

func deepGet(item interface{}, path string) interface{} {
	if path == "" {
		return item
	}

	parts := strings.Split(path, ".")
	itemValue := reflect.ValueOf(item)

	if len(parts) > 0 {
		switch itemValue.Kind() {
		case reflect.Struct:
			fieldValue := itemValue.FieldByName(parts[0])
			if fieldValue.IsValid() {
				return deepGet(fieldValue.Interface(), strings.Join(parts[1:], "."))
			}
		case reflect.Map:
			mapValue := itemValue.MapIndex(reflect.ValueOf(parts[0]))
			if mapValue.IsValid() {
				return deepGet(mapValue.Interface(), strings.Join(parts[1:], "."))
			}
		default:
			fmt.Printf("can't group by %s\n", path)
		}
		return nil
	}

	return itemValue.Interface()
}

func groupBy(entries []*RuntimeContainer, key string) map[string][]*RuntimeContainer {
	groups := make(map[string][]*RuntimeContainer)
	for _, v := range entries {
		value := deepGet(*v, key)
		if value != nil {
			groups[value.(string)] = append(groups[value.(string)], v)
		}
	}
	return groups
}

func contains(a map[string]string, b string) bool {
	if _, ok := a[b]; ok {
		return true
	}
	return false
}

func usage() {
	println("Usage: docker-gen [-watch=false] [-notify=\"restart xyz\"] <template> [<dest>]")
}

func generateFile(templatePath string, containers []*RuntimeContainer) {
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"contains": contains,
		"groupBy":  groupBy,
	}).ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}

	tmpl = tmpl
	dest := os.Stdout
	if flag.NArg() == 2 {
		dest, err = os.Create(flag.Arg(1))
		if err != nil {
			fmt.Println("unable to create dest file %s: %s\n", flag.Arg(1), err)
			os.Exit(1)
		}
	}

	err = tmpl.ExecuteTemplate(dest, filepath.Base(templatePath), containers)
	if err != nil {
		fmt.Printf("template error: %s\n", err)
	}
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

		if !onlyExposed {
			containers = append(containers, runtimeContainer)
			continue
		}

		if onlyExposed && len(runtimeContainer.Addresses) > 0 {
			containers = append(containers, runtimeContainer)
		}

	}

	generateFile(flag.Arg(0), containers)
}

func runNotifyCmd() {
	if notifyCmd == "" {
		return
	}

	args := strings.Split(notifyCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("error running notify command: %s\n", err)
	}
	print(string(output))

}

func main() {
	flag.BoolVar(&watch, "watch", false, "watch for container changes")
	flag.BoolVar(&onlyExposed, "only-exposed", false, "only include containers with exposed ports")
	flag.StringVar(&notifyCmd, "notify", "", "run command after template is regenerated")
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)

	if err != nil {
		panic(err)
	}

	generateFromContainers(client)
	runNotifyCmd()
	if !watch {
		return
	}

	eventChan := getEvents()
	for {
		event := <-eventChan
		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			generateFromContainers(client)
			runNotifyCmd()
		}
	}

}
