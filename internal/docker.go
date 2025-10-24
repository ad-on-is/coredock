package internal

import (
	"fmt"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/thoas/go-funk"
)

type DockerClient struct {
	client  *docker.Client
	channel chan *Service
	config  *Config
}

func NewDockerClient(channel chan *Service, conf *Config) (*DockerClient, error) {
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: client, channel: channel, config: conf}, nil
}

func (d *DockerClient) Run() error {
	dockerChan := make(chan *docker.APIEvents)

	d.client.AddEventListener(dockerChan)

	containers, err := d.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return fmt.Errorf("error getting containers: %w", err)
	}

	for _, c := range containers {
		d.channel <- NewService(&c, "start", d.config)
	}

	for e := range dockerChan {
		d.handleService(e)
	}
	return nil
}

func (d *DockerClient) findContainer(id string) (*docker.APIContainers, error) {
	containers, err := d.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("error getting containers: %w", err)
	}

	for _, c := range containers {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("container '%s' not found", shortContainerID(id))
}

func (d *DockerClient) handleService(e *docker.APIEvents) {
	actions := []string{"create", "start", "stop", "connect", "disconnect", "destroy"}

	if !funk.Contains(actions, e.Action) {
		return
	}

	id := e.ID
	if e.Action == "connect" || e.Action == "disconnect" {
		id = e.Actor.Attributes["container"]
	}

	c, err := d.findContainer(id)
	if err != nil {
		logger.Debugf("Could not handle service for actcion '%s': %v", e.Action, err)
		return
	}
	d.channel <- NewService(c, e.Action, d.config)
}

func cleanContainerName(name string) string {
	return strings.ReplaceAll(name, "/", "")
}

func shortContainerID(id string) string {
	if len(id) < 10 {
		return id
	}
	return id[:10]
}
