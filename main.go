package main

import (
	"encoding/json"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/gorhill/cronexpr"
	"github.com/palicao/docker-executor/lib"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"flag"
)

type ApiResponse struct {
	JobName   string
	StartTime time.Time
	EndTime   time.Time
	Output    []string
}

func main() {
	configFile := flag.String("config", "./config.yaml", "specify the yaml config file location")
	flag.Parse()

	config, err := lib.GetConfigFromFile(*configFile)
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}

	api := lib.NewDockerApi(cli)

	done := make(chan bool)
	go startServer(config, api, done)
	go scheduleJobs(config, api, done)

	<-done
}

func scheduleJobs(config *lib.Config, api *lib.DockerApi, done chan bool) {
	var wg sync.WaitGroup
	for _, job := range config.Jobs {
		if job.Schedule != "" {
			scheduleJob(job, api, &wg)
		}
	}
	wg.Wait()
	done <- true
}

func startServer(config *lib.Config, api *lib.DockerApi, done chan bool) {
	for jobName, job := range config.Jobs {
		if job.ApiExpose == true {
			http.HandleFunc("/jobs/run/"+jobName, func(w http.ResponseWriter, r *http.Request) {

				startTime := time.Now()
				response := runJob(job, api)
				endTime := time.Now()

				res := ApiResponse{
					JobName:   jobName,
					StartTime: startTime,
					EndTime:   endTime,
					Output:    prepareOutput(response),
				}

				js, err := json.Marshal(res)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
			})
		}
	}
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("error starting http server: %v", err)
	}
	done <- true
}

func prepareOutput(in []byte) []string {
	s := strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, string(in))
	return strings.Split(strings.Trim(s, "\n"), "\n")
}

func runJob(job lib.Job, api *lib.DockerApi) []byte {
	fmt.Printf("running job %s\n", time.Now().Format("15:04:05"))
	if job.Type == lib.JobTypeRun {
		response, err := api.RunJobAsContainer(job)
		if err != nil {
			log.Fatalf("error running container: %v", err)
		}
		return response
	} else {
		response, err := api.RunJobAsService(job)
		if err != nil {
			log.Fatalf("error running service: %v", err)
		}
		return response
	}
}

func scheduleJob(job lib.Job, api *lib.DockerApi, wg *sync.WaitGroup) {
	nextTime := cronexpr.MustParse(job.Schedule).Next(time.Now())
	wg.Add(1)
	go func() {
		time.Sleep(nextTime.Sub(time.Now()))
		scheduleJob(job, api, wg)
		response := runJob(job, api)
		fmt.Println(string(response))
		wg.Done()
	}()
}
