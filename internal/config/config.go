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
	OnlyExposed            bool
	OnlyPublished          bool
	ContainerFilter        map[string][]string
	Interval               int
	KeepBlankLines         bool
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
