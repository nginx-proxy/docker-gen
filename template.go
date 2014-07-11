package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func groupByMulti(entries []*RuntimeContainer, key, sep string) map[string][]*RuntimeContainer {
	groups := make(map[string][]*RuntimeContainer)
	for _, v := range entries {
		value := deepGet(*v, key)
		if value != nil {
			items := strings.Split(value.(string), sep)
			for _, item := range items {
				groups[item] = append(groups[item], v)
			}

		}
	}
	return groups
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

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func generateFile(config Config, containers []*RuntimeContainer) bool {
	templatePath := config.Template
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"contains":     contains,
		"exists":       exists,
		"groupBy":      groupBy,
		"groupByMulti": groupByMulti,
		"split":        strings.Split,
	}).ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	filteredContainers := []*RuntimeContainer{}
	if config.OnlyPublished {
		for _, container := range containers {
			if len(container.PublishedAddresses()) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else if config.OnlyExposed {
		for _, container := range containers {
			if len(container.Addresses) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else {
		filteredContainers = containers
	}

	dest := os.Stdout
	if config.Dest != "" {
		dest, err = ioutil.TempFile(filepath.Dir(config.Dest), "docker-gen")
		defer func() {
			dest.Close()
			os.Remove(dest.Name())
		}()
		if err != nil {
			log.Fatalf("unable to create temp file: %s\n", err)
		}
	}

	var buf bytes.Buffer
	multiwriter := io.MultiWriter(dest, &buf)
	err = tmpl.ExecuteTemplate(multiwriter, filepath.Base(templatePath), filteredContainers)
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	if config.Dest != "" {

		contents := []byte{}
		if _, err := os.Stat(config.Dest); err == nil {
			contents, err = ioutil.ReadFile(config.Dest)
			if err != nil {
				log.Fatalf("unable to compare current file contents: %s: %s\n", config.Dest, err)
			}
		}

		if bytes.Compare(contents, buf.Bytes()) != 0 {
			err = os.Rename(dest.Name(), config.Dest)
			if err != nil {
				log.Fatalf("unable to create dest file %s: %s\n", config.Dest, err)
			}
			log.Printf("Generated '%s' from %d containers", config.Dest, len(filteredContainers))
			return true
		}
		return false
	}
	return true
}
