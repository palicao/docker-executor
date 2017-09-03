package main

import (
	"fmt"
	"log"
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

		if jobConf.Type == "run" {
			response, err := api.RunJobAsContainer(jobConf)
			if err != nil {
				log.Fatalf("Error running container: %v", err)
			}
			fmt.Println(string(response))
		} else {
			response, err := api.RunJobAsService(jobConf)
			if err != nil {
				log.Fatalf("Error running service: %v", err)
			}
			fmt.Println(string(response))
		}
	}
}
