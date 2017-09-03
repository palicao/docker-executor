package lib

import (
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type DockerApi struct {
	client *client.Client
}

func NewDockerClient(cli *client.Client) *DockerApi {
	return &DockerApi{client: cli}
}

func (api *DockerApi) imageExists(ctx context.Context, image string, tag string) (result bool, err error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", image+":"+tag)
	options := types.ImageListOptions{Filters: filterArgs}
	images, err := api.client.ImageList(ctx, options)
	if err != nil {
		return false, err
	}
	return len(images) == 1, nil
}

func (api *DockerApi) pullImage(ctx context.Context, image string, tag string) error {
	response, err := api.client.ImagePull(ctx, image+":"+tag, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer response.Close()

	_, err = ioutil.ReadAll(response)
	if err != nil {
		return err
	}
	return nil
}

func (api *DockerApi) isTaskComplete(ctx context.Context, serviceId string, doneC chan bool, errC chan error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("service", serviceId)
	tasks, err := api.client.TaskList(ctx, types.TaskListOptions{Filters: filterArgs})
	if err != nil {
		errC <- err
	}

	if len(tasks) != 1 {
		errC <- errors.Errorf("Unable to inspect tasks for service %s", serviceId)
	}

	task := tasks[0]

	switch task.Status.State {
	case swarm.TaskStateFailed:
		errC <- errors.Errorf("Service failed")
	case swarm.TaskStateRejected:
		errC <- errors.Errorf("Service rejected")
	case swarm.TaskStateComplete:
		doneC <- true
	}
}

func (api *DockerApi) taskWait(ctx context.Context, serviceId string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	doneC := make(chan bool, 1)
	errC := make(chan error, 1)
	for {
		select {
		case <-ticker.C:
			api.isTaskComplete(ctx, serviceId, doneC, errC)
		case e := <-errC:
			return e
		case <-doneC:
			return nil
		}
	}
}

func (api *DockerApi) RunJobAsContainer(job Job) (out []byte, err error) {

	ctx := context.Background()

	imageExists, err := api.imageExists(ctx, job.Image, job.Tag)
	if err != nil {
		return nil, err
	}

	if !imageExists {
		err = api.pullImage(ctx, job.Image, job.Tag)
		if err != nil {
			return nil, err
		}
	}

	createResponse, err := api.client.ContainerCreate(ctx, &container.Config{
		Image: job.Image,
		Cmd:   job.Cmd,
		Env:   job.Env,
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}

	err = api.client.ContainerStart(ctx, createResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	resC, errC := api.client.ContainerWait(ctx, createResponse.ID, container.WaitConditionNextExit)
	select {
	case <-resC:
		break
	case err := <-errC:
		return nil, err
	}

	logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	logResponse, err := api.client.ContainerLogs(ctx, createResponse.ID, logOptions)
	if err != nil {
		return nil, err
	}
	defer logResponse.Close()

	response, err := ioutil.ReadAll(logResponse)
	if err != nil {
		return nil, err
	}

	api.client.ContainerRemove(ctx, createResponse.ID, types.ContainerRemoveOptions{})

	return response, nil
}

func (api *DockerApi) RunJobAsService(job Job) (out []byte, err error) {

	ctx := context.Background()

	replicas := uint64(1)
	replicatedOptions := &swarm.ReplicatedService{
		Replicas: &replicas,
	}

	containerSpec := &swarm.ContainerSpec{
		Image:   job.Image,
		Command: job.Cmd,
		Env:     job.Env,
		//TODO secrets, configs
	}

	placementPreferences := []swarm.PlacementPreference{}
	for _, p := range job.PlacementPreferences {
		preference := swarm.PlacementPreference{
			Spread: &swarm.SpreadOver{SpreadDescriptor: p},
		}
		placementPreferences = append(placementPreferences, preference)
	}

	placement := &swarm.Placement{
		Constraints: job.Constraints,
		Preferences: placementPreferences,
	}

	taskTemplate := swarm.TaskSpec{
		ContainerSpec: containerSpec,
		RestartPolicy: &swarm.RestartPolicy{Condition: swarm.RestartPolicyConditionNone},
		Placement:     placement,
	}

	createResponse, err := api.client.ServiceCreate(ctx, swarm.ServiceSpec{
		Mode:         swarm.ServiceMode{Replicated: replicatedOptions},
		TaskTemplate: taskTemplate,
	}, types.ServiceCreateOptions{})
	if err != nil {
		return nil, err
	}

	err = api.taskWait(ctx, createResponse.ID)
	if err != nil {
		return nil, err
	}

	logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	logResponse, err := api.client.ServiceLogs(ctx, createResponse.ID, logOptions)
	if err != nil {
		return nil, err
	}
	defer logResponse.Close()

	response, err := ioutil.ReadAll(logResponse)
	if err != nil {
		return nil, err
	}

	err = api.client.ServiceRemove(ctx, createResponse.ID)
	if err != nil {
		return nil, err
	}

	return response, nil
}
