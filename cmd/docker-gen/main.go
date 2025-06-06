package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/nginx-proxy/docker-gen/internal/config"
	"github.com/nginx-proxy/docker-gen/internal/generator"
)

type stringslice []string
type mapstringslice map[string][]string

var (
	buildVersion          string
	version               bool
	watch                 bool
	wait                  string
	notifyCmd             string
	notifyOutput          bool
	sighupContainerID     stringslice
	notifyContainerID     stringslice
	notifyContainerSignal int
	notifyContainerFilter mapstringslice = make(mapstringslice)
	onlyExposed           bool
	onlyPublished         bool
	includeStopped        bool
	configFiles           stringslice
	configs               config.ConfigFile
	eventFilter           mapstringslice = mapstringslice{"event": {"start", "stop", "die", "health_status"}}
	interval              int
	keepBlankLines        bool
	endpoint              string
	tlsCert               string
	tlsKey                string
	tlsCaCert             string
	tlsVerify             bool
)

func (strings *stringslice) String() string {
	return "[]"
}

func (strings *stringslice) Set(value string) error {
	*strings = append(*strings, value)
	return nil
}

func (filter *mapstringslice) String() string {
	return "[string][]string"
}

func (filter *mapstringslice) Set(value string) error {
	name, value, found := strings.Cut(value, "=")
	if found {
		(*filter)[name] = append((*filter)[name], value)
	}
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
	println(`For more information, see https://github.com/nginx-proxy/docker-gen`)
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
	flag.Var(&sighupContainerID, "notify-sighup",
		"send HUP signal to container. Equivalent to docker kill -s HUP `container-ID`. You can pass this option multiple times to send HUP to multiple containers.")
	flag.Var(&notifyContainerID, "notify-container",
		"send -notify-signal signal (defaults to 1 / HUP) to container. You can pass this option multiple times to notify multiple containers.")
	flag.Var(&notifyContainerFilter, "notify-filter",
		"container filter for notification (e.g -notify-filter name=foo). You can pass this option multiple times to combine filters with AND. https://docs.docker.com/engine/reference/commandline/ps/#filter")
	flag.IntVar(&notifyContainerSignal, "notify-signal", int(docker.SIGHUP),
		"signal to send to the notify-container and notify-filter. Defaults to SIGHUP")
	flag.Var(&configFiles, "config", "config files with template directives. Config files will be merged if this option is specified multiple times.")
	flag.IntVar(&interval, "interval", 0, "notify command interval (secs)")
	flag.BoolVar(&keepBlankLines, "keep-blank-lines", false, "keep blank lines in the output file")
	flag.StringVar(&endpoint, "endpoint", "", "docker api endpoint (tcp|unix://..). Default unix:///var/run/docker.sock")
	flag.StringVar(&tlsCert, "tlscert", filepath.Join(certPath, "cert.pem"), "path to TLS client certificate file")
	flag.StringVar(&tlsKey, "tlskey", filepath.Join(certPath, "key.pem"), "path to TLS client key file")
	flag.StringVar(&tlsCaCert, "tlscacert", filepath.Join(certPath, "ca.pem"), "path to TLS CA certificate file")
	flag.BoolVar(&tlsVerify, "tlsverify", os.Getenv("DOCKER_TLS_VERIFY") != "", "verify docker daemon's TLS certicate")

	flag.Var(&eventFilter, "event-filter",
		"additional filter for event watched by docker-gen (e.g -event-filter event=connect -event-filter event=disconnect). You can pass this option multiple times to combine filters. By default docker-gen listen for container events start, stop, die and health_status. https://docs.docker.com/engine/reference/commandline/events/#filtering-events")

	flag.Usage = usage
	flag.Parse()
}

func main() {
	// SIGHUP is used to trigger generation but go programs call os.Exit(2) at default.
	// Ignore the signal until the handler is registered:
	signal.Ignore(syscall.SIGHUP)

	initFlags()

	if version {
		fmt.Println(buildVersion)
		return
	}

	if flag.NArg() < 1 && len(configFiles) == 0 {
		usage()
		os.Exit(1)
	}

	slices.Sort(configFiles)
	configFiles = slices.Compact(configFiles)

	if len(configFiles) > 0 {
		for _, configFile := range configFiles {
			err := loadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config %s: %s\n", configFile, err)
			}
		}
	} else {
		w, err := config.ParseWait(wait)
		if err != nil {
			log.Fatalf("Error parsing wait interval: %s\n", err)
		}
		cfg := config.Config{
			Template:         flag.Arg(0),
			Dest:             flag.Arg(1),
			Watch:            watch,
			Wait:             w,
			NotifyCmd:        notifyCmd,
			NotifyOutput:     notifyOutput,
			NotifyContainers: make(map[string]int),
			OnlyExposed:      onlyExposed,
			OnlyPublished:    onlyPublished,
			IncludeStopped:   includeStopped,
			Interval:         interval,
			KeepBlankLines:   keepBlankLines,
		}
		for _, id := range sighupContainerID {
			cfg.NotifyContainers[id] = int(syscall.SIGHUP)
		}
		for _, id := range notifyContainerID {
			cfg.NotifyContainers[id] = notifyContainerSignal
		}
		if len(notifyContainerFilter) > 0 {
			cfg.NotifyContainersFilter = notifyContainerFilter
			cfg.NotifyContainersSignal = notifyContainerSignal
		}
		configs = config.ConfigFile{
			Config: []config.Config{cfg},
		}
	}

	all := false
	for _, config := range configs.Config {
		if config.IncludeStopped {
			all = true
		}
	}

	generator, err := generator.NewGenerator(generator.GeneratorConfig{
		Endpoint:    endpoint,
		TLSKey:      tlsKey,
		TLSCert:     tlsCert,
		TLSCACert:   tlsCaCert,
		TLSVerify:   tlsVerify,
		All:         all,
		EventFilter: eventFilter,
		ConfigFile:  configs,
	})

	if err != nil {
		log.Fatalf("Error creating generator: %v", err)
	}

	if err := generator.Generate(); err != nil {
		log.Fatalf("Error running generate: %v", err)
	}
}
