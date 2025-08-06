package config

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

type CamCfg struct {
	Name      string `yaml:"name"`
	URL       string `yaml:"url"`
	Duration  string `yaml:"duration"`
	Transport string `yaml:"transport,omitempty"`
	Slack     string `yaml:"slack,omitempty"`
}

func LoadConfig(path string) ([]CamCfg, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct { Cameras []CamCfg `yaml:"cameras"` }
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	for i := range raw.Cameras {
		c := &raw.Cameras[i]
		if c.Transport == "" {
			c.Transport = "udp"
		}
		if c.Slack == "" {
			c.Slack = "10s"
		}
		if _, err := time.ParseDuration(c.Duration); err != nil {
			return nil, fmt.Errorf("camera %s: invalid duration '%s': %w", c.Name, c.Duration, err)
		}
		if _, err := time.ParseDuration(c.Slack); err != nil {
			return nil, fmt.Errorf("camera %s: invalid slack '%s': %w", c.Name, c.Slack, err)
		}
		if !isValidName(c.Name) {
			return nil, fmt.Errorf("camera name '%s' contains invalid characters", c.Name)
		}
	}

	return raw.Cameras, nil
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
func isValidName(n string) bool {
	return nameRe.MatchString(n)
}
