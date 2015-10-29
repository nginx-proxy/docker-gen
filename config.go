package dockergen

import "github.com/fsouza/go-dockerclient"

type Config struct {
	Template         string
	Dest             string
	Watch            bool
	NotifyCmd        string
	NotifyOutput     bool
	NotifyContainers map[string]docker.Signal
	OnlyExposed      bool
	OnlyPublished    bool
	IncludeStopped   bool
	Interval         int
	KeepBlankLines   bool
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
