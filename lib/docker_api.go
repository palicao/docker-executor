package lib

import (
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types"
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

func (api *DockerApi) RunContainer(job Job) (rc io.ReadCloser, err error) {

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
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}

	err = api.client.ContainerStart(ctx, createResponse.ID, types.ContainerStartOptions{});
	if err != nil {
		return nil, err
	}

	resC, errC := api.client.ContainerWait(ctx, createResponse.ID, container.WaitConditionNextExit)
	select {
	case <- resC:
		break
	case err := <-errC:
		return nil, err
	}

	logResponse, err := api.client.ContainerLogs(ctx, createResponse.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return nil, err
	}

	api.client.ContainerRemove(ctx, createResponse.ID, types.ContainerRemoveOptions{})

	return logResponse, nil
}

func (api *DockerApi) RunService(service string) (rc io.ReadCloser, err error) {
	return nil, nil
}
