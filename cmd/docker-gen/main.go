package main

import (
	"flag"
	"fmt"
	"github.com/thoas/go-funk"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/jwilder/docker-gen"
)

type stringslice []string

var (
	buildVersion            string
	version                 bool
	swarmMode               bool
	watch                   bool
	wait                    string
	notifyCmd               string
	notifyOutput            bool
	notifySigHUPContainerID string
	onlyExposed             bool
	onlyPublished           bool
	includeStopped          bool
	configFiles             stringslice
	configs                 dockergen.ConfigFile
	interval                int
	keepBlankLines          bool
	endpoints               stringslice
	tlsCerts                stringslice
	tlsKeys                 stringslice
	tlsCaCerts              stringslice
	tlsCertPaths            stringslice
	tlsVerify               bool
	wg                      sync.WaitGroup
)

func (strings *stringslice) String() string {
	return "[]"
}

func (strings *stringslice) Set(value string) error {
	// TODO: Throw an error for duplicate `dest`
	*strings = append(*strings, value)
	return nil
}

func usage() {
	println(`Usage: docker-gen [options] template [dest]

Generate files from docker container meta-data

Options:`)
	flag.PrintDefaults()

	println(`
Arguments:
  template - path to a template to generate
  dest - path to a write the template.  If not specfied, STDOUT is used`)

	println(`
Environment Variables:
  DOCKER_HOST - default value for -endpoint
  DOCKER_CERT_PATH - directory path containing key.pem, cert.pem and ca.pem
  DOCKER_TLS_VERIFY - enable client TLS verification
`)
	println(`For more information, see https://github.com/jwilder/docker-gen`)
}

func loadConfig(file string) error {
	_, err := toml.DecodeFile(file, &configs)
	if err != nil {
		return err
	}
	return nil
}

func initFlags() {

	certPath := filepath.Join(os.Getenv("DOCKER_CERT_PATH"))
	if certPath == "" {
		certPath = filepath.Join(os.Getenv("HOME"), ".docker")
	}
	flag.BoolVar(&version, "version", false, "show version")
	flag.BoolVar(&watch, "watch", false, "watch for container changes")
	flag.StringVar(&wait, "wait", "", "minimum and maximum durations to wait (e.g. \"500ms:2s\") before triggering generate")
	flag.BoolVar(&onlyExposed, "only-exposed", false, "only include containers with exposed ports")

	flag.BoolVar(&onlyPublished, "only-published", false,
		"only include containers with published ports (implies -only-exposed)")
	flag.BoolVar(&includeStopped, "include-stopped", false, "include stopped containers")
	flag.BoolVar(&notifyOutput, "notify-output", false, "log the output(stdout/stderr) of notify command")
	flag.StringVar(&notifyCmd, "notify", "", "run command after template is regenerated (e.g `restart xyz`)")
	flag.StringVar(&notifySigHUPContainerID, "notify-sighup", "",
		"send HUP signal to container.  Equivalent to docker kill -s HUP `container-ID`")
	flag.Var(&configFiles, "config", "config files with template directives. Config files will be merged if this option is specified multiple times.")
	flag.IntVar(&interval, "interval", 0, "notify command interval (secs)")
	flag.BoolVar(&keepBlankLines, "keep-blank-lines", false, "keep blank lines in the output file")

	flag.BoolVar(&swarmMode, "swarmMode", false, "Enable Swarm Mode, multiple nodes")
	flag.Var(&tlsCertPaths, "tlsCertPaths", "folder store docker host certs and keys")
	flag.Var(&endpoints, "endpoints", "docker api endpoint (tcp|unix://..). Default unix:///var/run/docker.sock")
	flag.Var(&tlsCerts, "tlscerts", "path to TLS client certificate file (cert.pem)")
	flag.Var(&tlsKeys, "tlskeys", "path to TLS client key file (key.pem)")
	flag.Var(&tlsCaCerts, "tlscacerts", "path to TLS CA certificate file (ca.pem)")
	flag.BoolVar(&tlsVerify, "tlsverify", os.Getenv("DOCKER_TLS_VERIFY") != "", "verify docker daemon's TLS certicate")

	flag.Usage = usage
	flag.Parse()
}

func main() {
	initFlags()

	if version {
		fmt.Println(buildVersion)
		return
	}

	if flag.NArg() < 1 && len(configFiles) == 0 {
		usage()
		os.Exit(1)
	}

	if len(configFiles) > 0 {
		for _, configFile := range configFiles {
			err := loadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config %s: %s\n", configFile, err)
			}
		}
	} else {
		w, err := dockergen.ParseWait(wait)
		if err != nil {
			log.Fatalf("Error parsing wait interval: %s\n", err)
		}
		config := dockergen.Config{
			Template:         flag.Arg(0),
			Dest:             flag.Arg(1),
			Watch:            watch,
			Wait:             w,
			NotifyCmd:        notifyCmd,
			NotifyOutput:     notifyOutput,
			NotifyContainers: make(map[string]docker.Signal),
			OnlyExposed:      onlyExposed,
			OnlyPublished:    onlyPublished,
			IncludeStopped:   includeStopped,
			Interval:         interval,
			KeepBlankLines:   keepBlankLines,
		}
		if notifySigHUPContainerID != "" {
			config.NotifyContainers[notifySigHUPContainerID] = docker.SIGHUP
		}
		configs = dockergen.ConfigFile{
			Config: []dockergen.Config{config},
		}
	}

	all := true
	for _, config := range configs.Config {
		if config.IncludeStopped {
			all = true
		}
	}

	if swarmMode {
		tlsCerts = funk.Map(tlsCertPaths, func(certPath string) string {
			return filepath.Join(certPath, "cert.pem")
		}).([]string)
		tlsKeys = funk.Map(tlsCertPaths, func(certPath string) string {
			return filepath.Join(certPath, "key.pem")
		}).([]string)
		tlsCaCerts = funk.Map(tlsCertPaths, func(certPath string) string {
			return filepath.Join(certPath, "ca.pem")
		}).([]string)
	}

	for i, _ := range endpoints {
		generator, err := dockergen.NewGenerator(dockergen.GeneratorConfig{
			Endpoint:   endpoints[i],
			TLSKey:     tlsKeys[i],
			TLSCert:    tlsCerts[i],
			TLSCACert:  tlsCaCerts[i],
			TLSVerify:  tlsVerify,
			All:        all,
			ConfigFile: configs,
		})

		if err != nil {
			log.Fatalf("Error creating generator: %v", err)
		}

		var generate = func() {
			if err := generator.Generate(); err != nil {
				log.Fatalf("Error running generate: %v", err)
			}
		}

		if i == len(endpoints)-1 {
			generate()
		} else {
			go generate()
		}
	}
}
