package internal

import (
	"fmt"
	"net"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

type Container struct {
	ID   string
	Name string
	IPs  []net.IP
	Port int
	SRVs map[string]int
}

type DockerClient struct {
	client     *docker.Client
	containers []docker.APIContainers
	channel    chan Container
}

func NewDockerClient(channel chan Container) (*DockerClient, error) {
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: client, channel: channel}, nil
}

func (d *DockerClient) Run() error {
	messageChan := make(chan *docker.APIEvents)

	d.client.AddEventListener(messageChan)

	containers, err := d.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return fmt.Errorf("error getting containers: %w", err)
	}
	d.containers = containers

	for _, container := range containers {
		c := Container{}
		c.Name = cleanContainerName(container.Names[0])
		d.channel <- c
	}

	for m := range messageChan {
		d.handleContainers(m)
	}
	return nil
}

func (d *DockerClient) handleContainers(e *docker.APIEvents) {
	id := e.ID
	var c *docker.APIContainers

	for _, container := range d.containers {
		if container.ID == id {
			c = &container
			break
		}
	}
	if c != nil {
		d.channel <- Container{
			ID:   c.ID,
			Name: cleanContainerName(c.Names[0]),
		}
	}
	fmt.Println("Docker event:", e.Action)
}

func cleanContainerName(name string) string {
	return strings.ReplaceAll(name, "/", "")
}
