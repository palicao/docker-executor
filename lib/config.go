package lib

import (
	"fmt"
	"io/ioutil"

	"github.com/gorhill/cronexpr"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Job struct {
	Type                 string   `yaml:"type"`
	Image                string   `yaml:"image"`
	Tag                  string   `yaml:"tag"`
	Service              string   `yaml:"service"`
	Schedule             string   `yaml:"schedule"`
	Secrets              []string `yaml:"secrets"`
	Configs              []string `yaml:"configs"`
	Cmd                  []string `yaml:"cmd"`
	Env                  []string `yaml:"env"`
	Constraints          []string `yaml:"constraints"`
	PlacementPreferences []string `yaml:"placement_preferences"`
	ApiExpose            bool     `yaml:"api_expose"`
}

type Config struct {
	Jobs map[string]Job `yaml:"jobs"`
}

func validateJob(job Job) error {
	if job.Type != "run" && job.Type != "service" {
		return errors.New("Type can only be run or service")
	}

	if job.Image == "" {
		return errors.New("Image must not be empty")
	}

	if job.Schedule != "" {
		_, err := cronexpr.Parse(job.Schedule)
		if err != nil {
			return errors.New("Schedule must be a valid cron expression")
		}
	}

	if job.Type == "run" && (len(job.Secrets) > 0 || len(job.Configs) > 0 || len(job.Constraints) > 0 || len(job.PlacementPreferences) > 0) {
		return errors.New("Secrets, configs, constraint and placement preferences are only allowed for services")
	}

	return nil
}

func prepareJob(job Job) Job {
	if job.Tag == "" {
		job.Tag = "latest"
	}
	return job
}

func CreateConfigFromFile(filename string) (config Config, err error) {
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("Unable to read file %s: %v", filename, err)
	}
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return config, fmt.Errorf("Unable to unmarshall file %s: %v", filename, err)
	}
	for i, j := range config.Jobs {
		err = validateJob(j)
		if err != nil {
			return config, fmt.Errorf("Configuration for job %s not valid: %v", i, err)
		}
		config.Jobs[i] = prepareJob(j)
	}
	return config, nil
}
