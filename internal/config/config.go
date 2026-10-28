package config

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	Template               string
	Dest                   string
	Watch                  bool
	Wait                   *Wait
	NotifyCmd              string
	NotifyOutput           bool
	NotifyContainers       map[string]int
	NotifyContainersFilter map[string][]string
	NotifyContainersSignal int
	ContainerFilter        map[string][]string
	IncludeStopped         bool
	OnlyExposed            bool
	OnlyPublished          bool
	Interval               int
	KeepBlankLines         bool
}

// SetContainerFilterDefaults adds to ContainerFilter the filters implied by IncludeStopped,
// OnlyPublished and OnlyExposed, unless ContainerFilter already has a filter for the same key.
func (c *Config) SetContainerFilterDefaults() {
	if c.ContainerFilter == nil {
		c.ContainerFilter = make(map[string][]string)
	}
	if len(c.ContainerFilter["status"]) == 0 {
		if c.IncludeStopped {
			c.ContainerFilter["status"] = []string{
				"created",
				"restarting",
				"running",
				"removing",
				"paused",
				"exited",
				"dead",
			}
		} else {
			c.ContainerFilter["status"] = []string{"running"}
		}
	}
	if c.OnlyPublished && len(c.ContainerFilter["publish"]) == 0 {
		c.ContainerFilter["publish"] = []string{"1-65535"}
	} else if c.OnlyExposed && len(c.ContainerFilter["expose"]) == 0 {
		c.ContainerFilter["expose"] = []string{"1-65535"}
	}
}

type ConfigFile struct {
	Config []Config
}

func (c *ConfigFile) FilterWatches() ConfigFile {
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

type Wait struct {
	Min time.Duration
	Max time.Duration
}

func (w *Wait) UnmarshalText(text []byte) error {
	wait, err := ParseWait(string(text))
	if err == nil {
		w.Min, w.Max = wait.Min, wait.Max
	}
	return err
}

func ParseWait(s string) (*Wait, error) {
	if len(strings.TrimSpace(s)) < 1 {
		return &Wait{0, 0}, nil
	}

	parts := strings.Split(s, ":")

	var (
		min time.Duration
		max time.Duration
		err error
	)
	min, err = time.ParseDuration(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, err
	}
	if len(parts) > 1 {
		max, err = time.ParseDuration(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		if max < min {
			return nil, errors.New("invalid wait interval: max must be larger than min")
		}
	} else {
		max = 4 * min
	}

	return &Wait{min, max}, nil
}
