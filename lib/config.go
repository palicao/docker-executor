package lib

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"fmt"
)

type Job struct {
	Type       string   `yaml:"type"`
	Image      string   `yaml:"image"`
	Tag        string   `yaml:"tag"`
	Service    string   `yaml:"service"`
	Entrypoint []string `yaml:"entrypoint"`
	Cmd        []string `yaml:"cmd"`
	Env        []string `yaml:"env"`
}

type Config struct {
	Jobs map[string]Job `yaml:"jobs"`
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
	return config, nil
}
