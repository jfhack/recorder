package config

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"
	"slices"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

type FFMPEGArgs struct {
	Global []string `yaml:"global,omitempty"`
	Input  []string `yaml:"input,omitempty"`
	Output []string `yaml:"output,omitempty"`
}

type CamCfg struct {
	Name      string     `yaml:"name"`
	URL       string     `yaml:"url"`
	Duration  string     `yaml:"duration"`
	AutoArgs  *bool      `yaml:"autoArgs,omitempty"`
	Slack     string     `yaml:"slack,omitempty"`
	Args      FFMPEGArgs `yaml:"args,omitempty"`
	Suffix    string     `yaml:"suffix,omitempty"`
}

func (c CamCfg) AutoArgsEnabled() bool {
	return c.AutoArgs == nil || *c.AutoArgs
}

func (c CamCfg) IsRTSP() bool {
	s := strings.TrimSpace(c.URL)
	u, err := url.Parse(s)
	if err == nil && u.Scheme != "" {
		return strings.EqualFold(u.Scheme, "rtsp") || strings.EqualFold(u.Scheme, "rtsps")
	}
	s = strings.ToLower(s)
	return strings.HasPrefix(s, "rtsp://") || strings.HasPrefix(s, "rtsps://")
}

func HasFlag(args []string, flag string) bool {
	return slices.Index(args, flag) >= 0
}

func HasCodecFlag(args []string) bool {
	for _, a := range args {
		la := strings.ToLower(a)
		if la == "-c" || la == "-codec" || strings.HasPrefix(la, "-c:") || strings.HasPrefix(la, "-codec:") {
			return true
		}
	}
	return false
}

func HasReencodeOnlyFlags(args []string) bool {
	reencode := []string{
		"-vf", "-af", "-filter", "-filter:v", "-filter:a", "-filter_complex",
		"-pix_fmt",
		"-r", "-s",
		"-b:v", "-b:a", "-maxrate", "-bufsize",
		"-crf", "-qp", "-q:v", "-aq",
		"-ar", "-ac", "-sample_fmt",
		"-profile:v", "-preset", "-tune", "-g",
	}
outer:
	for _, a := range args {
		la := strings.ToLower(a)
		for _, f := range reencode {
			if la == f {
				return true
			}
			if strings.HasSuffix(f, ":") {
				if strings.HasPrefix(la, f) { return true }
			} else if strings.HasSuffix(f, ":v") || strings.HasSuffix(f, ":a") {
				if la == f { return true }
			} else if strings.HasPrefix(la, f+":") {
				return true
			}
		}
		continue outer
	}
	return false
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
		if c.Slack == "" {
			c.Slack = "10s"
		}
		if c.Suffix == "" {
			c.Suffix = ".mkv"
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
