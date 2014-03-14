package main

import (
	"github.com/fsouza/go-dockerclient"
	"os"
	"path/filepath"
	"text/template"
)

func usage() {
	println("Usage: docker-log template.file")
}

func generateFile(templatePath string, containers []docker.APIContainers) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}

	err = tmpl.ExecuteTemplate(os.Stdout, filepath.Base(templatePath), containers)
}

func main() {

	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)

	if err != nil {
		panic(err)
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{
		All: false,
	})
	if err != nil {
		panic(err)
	}

	generateFile(os.Args[1], containers)
}
