package lib

import (
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
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

func (api *DockerApi) RunJobAsContainer(job Job) (rc io.ReadCloser, err error) {

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

	api.client.ContainerRemove(ctx, createResponse.ID, types.ContainerRemoveOptions{})

	return logResponse, nil
}

func (api *DockerApi) RunJobAsService(job Job) (rc io.ReadCloser, err error) {

	ctx := context.Background()

	replicas := uint64(1)
	replicatedOptions := swarm.ReplicatedService{
		Replicas: &replicas,
	}

	containerSpec := swarm.ContainerSpec{
		Image:   job.Image,
		Command: job.Cmd,
		Env:     job.Env,
		//TODO secrets, configs
	}

	taskTemplate := swarm.TaskSpec{
		ContainerSpec: &containerSpec,
	}

	resp, err := api.client.ServiceCreate(ctx, swarm.ServiceSpec{
		Mode:         swarm.ServiceMode{Replicated: &replicatedOptions},
		TaskTemplate: taskTemplate,
	}, types.ServiceCreateOptions{})
	if err != nil {
		return nil, err
	}

	logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	logResponse, err := api.client.ServiceLogs(ctx, resp.ID, logOptions)
	defer logResponse.Close()

	//filterArgs := filters.NewArgs()
	//filterArgs.Add("service", resp.ID)
	//tasks, err := api.client.TaskList(ctx, types.TaskListOptions{Filters: filterArgs})
	//if err != nil {
	//	return nil, err
	//}
	//task := tasks[0]
	//logResponse, err := api.client.TaskLogs(ctx, task.ID, logOptions)
	//if err != nil {
	//	return nil, err
	//}
	//defer logResponse.Close()

	api.client.ServiceRemove(ctx, resp.ID)

	return logResponse, nil
}
