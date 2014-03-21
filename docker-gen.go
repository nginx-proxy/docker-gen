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
	"os/signal"
	"path/filepath"
	"syscall"
	"text/template"
)

var watch bool

type Event struct {
	ContainerId string `json:"id"`
	Status      string `json:"status"`
	Image       string `json:"from"`
}

func usage() {
	println("Usage: docker-gen [-watch] <template> [<dest>]")
}

func generateFile(templatePath string, containers []docker.APIContainers) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}

	dest := os.Stdout
	if flag.NArg() == 2 {
		dest, err = os.Create(flag.Arg(1))
		if err != nil {
			fmt.Println("unable to create dest file %s: %s", flag.Arg(1), err)
			os.Exit(1)
		}
	}

	err = tmpl.ExecuteTemplate(dest, filepath.Base(templatePath), containers)
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

		c, err := newConn()
		if err != nil {
			fmt.Printf("cannot connect to docker: %s", err)
			return
		}
		defer c.Close()

		req, err := http.NewRequest("GET", "/events", nil)
		if err != nil {
			fmt.Printf("bad request for events: %s", err)
			return
		}

		resp, err := c.Do(req)
		if err != nil {
			fmt.Printf("cannot connect to events endpoint: %s", err)
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
				fmt.Printf("cannot decode json: %s", err)
				continue
			}
			eventChan <- event
		}
		fmt.Printf("closing event channel")
	}()
	return eventChan
}

func generateFromContainers(client *docker.Client) {
	containers, err := client.ListContainers(docker.ListContainersOptions{
		All: false,
	})
	if err != nil {
		fmt.Printf("error listing containers: %s", err)
		return
	}

	generateFile(flag.Arg(0), containers)
}

func main() {

	flag.BoolVar(&watch, "watch", false, "watch for container changes")
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
	if !watch {
		return
	}

	eventChan := getEvents()
	for {
		event := <-eventChan
		if event.Status == "start" || event.Status == "stop" {
			generateFromContainers(client)
		}
	}

}
