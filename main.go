package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/palicao/docker-executor/lib"
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

func main() {
	config, err := lib.CreateConfigFromFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	for name, jobConf := range config.Jobs {

		fmt.Printf("Executing job %s\n", name)

		cli, err := client.NewEnvClient()

		if err != nil {
			log.Fatalf("Error creating client: %v", err)
		}

		api := lib.NewDockerClient(cli)

		response, err := api.RunContainer(jobConf)
		if err != nil {
			log.Fatalf("Error running container: %v", err)
		}
		io.Copy(os.Stdout, response)
	}
}
