# Docker-Executor

## Disclaimer
**This is an experimental project, don't use it in production!**

## Motivation
Docker lacks the native ability to schedule and run one-shot jobs.

This project takes inspiration mostly from the work of [Alex Ellis](https://github.com/alexellis) and his
[Jaas](https://github.com/alexellis/jaas) and [Faas](https://github.com/alexellis/faas), but I wanted to keep
things very simple and minimal and learn some docker internals.

## Setup
Build the Docker-Executor container:

```
docker build . -t docker-executor
```

Run it exposing port 8080 and mounting the docker socket and a config file: done!

```
docker run \
    -p 8080:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v ./config.yaml:/etc/docker-executor/config.yaml \
    docker-executor
```
You can also run the same image in swarm mode with similar settings.

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
      - node.labels.type == oneshot
    placement_preferences: # only for services
      - spread=node.labels.datacenter
    api_expose: true # if you want to expose the service via the built-in API
```

## Api
You can run jobs by GETting `localhost:8080\jobs\run\job_name`.
You will get something like this as response:
```
{
    "JobName": "job_name",
    "StartTime": "2017-09-13T16:04:01.396146735Z",
    "EndTime": "2017-09-13T16:04:05.439377007Z",
    "Output": ["cache","empty","lib","local","lock","log","opt","run","spool","tmp"]
}
```

## Vendor folder
I had to mess around with the docker project source code because of the
current confusion in the project itself (docker vs. moby vs. docker-ce).

That's why I couldn't use any package manager and I had to include all the dependencies
in the vendor folder.

Hopefully this will change as soon as docker will finally fix the current situation.

## Todo
* Secrets, configs
* Tests :D
* Config hot reload
