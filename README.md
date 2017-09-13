# Docker-Executor

## Disclaimer
This is an experimental project, far from being ready!

## Motivation
Docker lacks the native ability to schedule and run one-shot jobs.

This project takes inspiration mostly from the work of [Alex Ellis](https://github.com/alexellis) and his
[Jaas](https://github.com/alexellis/jaas) and [Faas](https://github.com/alexellis/faas), but I wanted to keep
things very simple and minimal and learn some docker internals.

## Setup
Build the Docker-Executor container and run it mounting the docker socket and a config file, done!

`docker run -p 80:80 -v /var/run/docker.sock:/var/run/docker.sock -v ./config.yaml:/etc/docker-executor/config.yaml docker-executor`

Or, you can run the container in swarm mode and mount the config file as a config (hot reload is planned for a future version).

## Config file
The config.yaml looks like this:
```yml
jobs:
  job_name:
    type: run # "run" is for using docker run, "service" is if you want to run in swarm mode
    image: alpine
    tag: latest
    service: alpine # name of the service in case type = "service"
    schedule: "* * * * *" # cron syntax, if you want to execute the job at given intervals
    secrets:
      - source=secret_name,target=/etc/config/secret.yaml
    configs:
      - source=config_name,target=/etc/config/config.yaml
    cmd:
      - ls
      - /var
    env:
      - key=value
    constraints: # only for services
      - node.labels.type == queue
    placement_preferences: # only for services
      - spread=node.labels.datacenter
    api_expose: true # if you want to expose the service via the built-in API
```

## Vendor mess
I cannot use a package manager and I had to version the whole folder because of the current
mess in the docker project (docker vs. moby vs. docker-ce).

Hopefully this will change as soon as docker will finally fix the current chaos!

## Todo
* Dockerfile
* Flag for choosing the config file (defaults to /etc/docker-executor/config.yaml or current folder)
* Secrets, configs
* Tests :D
