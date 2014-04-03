package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

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

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func generateFile(config Config, containers []*RuntimeContainer) bool {
	templatePath := config.Template
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"contains": contains,
		"groupBy":  groupBy,
	}).ParseFiles(templatePath)
	if err != nil {
		panic(err)
	}

	filteredContainers := []*RuntimeContainer{}
	if config.OnlyExposed {
		for _, container := range containers {
			if len(container.Addresses) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else {
		filteredContainers = containers
	}

	tmpl = tmpl
	dest := os.Stdout
	if config.Dest != "" {
		dest, err = ioutil.TempFile("", "docker-gen")
		defer dest.Close()
		if err != nil {
			fmt.Printf("unable to create temp file: %s\n", err)
			os.Exit(1)
		}
	}

	var buf bytes.Buffer
	multiwriter := io.MultiWriter(dest, &buf)
	err = tmpl.ExecuteTemplate(multiwriter, filepath.Base(templatePath), containers)
	if err != nil {
		fmt.Printf("template error: %s\n", err)
	}

	if config.Dest != "" {

		contents := []byte{}
		if _, err := os.Stat(config.Dest); err == nil {
			contents, err = ioutil.ReadFile(config.Dest)
			if err != nil {
				fmt.Printf("unable to compare current file contents: %s: %s\n", config.Dest, err)
				os.Exit(1)
			}
		}

		if bytes.Compare(contents, buf.Bytes()) != 0 {
			err = os.Rename(dest.Name(), config.Dest)
			if err != nil {
				fmt.Printf("unable to create dest file %s: %s\n", config.Dest, err)
				os.Exit(1)
			}
			return true
		}
		return false
	}
	return true
}
