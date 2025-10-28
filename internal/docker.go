package internal

import (
	"fmt"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/thoas/go-funk"
)

type DockerClient struct {
	client  *docker.Client
	channel chan *[]Service
	config  *Config
}

func NewDockerClient(channel chan *[]Service, conf *Config) (*DockerClient, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: client, channel: channel, config: conf}, nil
}

func (d *DockerClient) sendContainers() {
	containers, err := d.getContainers()
	if err != nil {
		return
	}

	services := []Service{}
	for _, c := range containers {
		d.maybeConnectToNetwork(&c)
		services = append(services, *NewService(&c, "start", d.config))
	}

	d.channel <- &services
}

func (d *DockerClient) Run() error {
	dockerChan := make(chan *docker.APIEvents)

	d.sendContainers()

	err := d.client.AddEventListener(dockerChan)
	if err != nil {
		return fmt.Errorf("error adding Docker event listener: %w", err)
	}

	for e := range dockerChan {
		logger.Debugf("Received event from Docker: %v", e)
		actions := []string{"create", "start", "stop", "connect", "disconnect", "destroy"}

		if funk.Contains(actions, e.Action) {
			d.sendContainers()
		}

	}
	return nil
}

func (d *DockerClient) getContainers() ([]docker.APIContainers, error) {
	containers, err := d.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("error getting containers: %w", err)
	}
	return funk.Filter(containers, func(c docker.APIContainers) bool {
		labels := c.Labels
		_, isIgnored := labels["coredock.ignore"]
		if isIgnored {
			logger.Debugf("Ignoring container '%s' due to 'coredock.ignore' label", cleanContainerName(c.Names[0]))
		}

		isCoredock := strings.Contains(c.Image, "coredock")

		return !isIgnored && !isCoredock
	}).([]docker.APIContainers), nil
}

func (d *DockerClient) maybeConnectToNetwork(c *docker.APIContainers) {
	for _, nw := range d.config.Networks {
		dnw, err := d.findNetwork(nw)
		if err != nil {
			logger.Errorf("Error finding network '%s': %v", nw, err)
			continue
		}
		logger.Debugf("Connected '%s' to network '%s'", cleanContainerName(c.Names[0]), dnw.Name)
		err = d.client.ConnectNetwork(dnw.ID, docker.NetworkConnectionOptions{Container: c.ID})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			logger.Errorf("Error connecting container '%s' to network '%s': %v", cleanContainerName(c.Names[0]), dnw.Name, err)
		}
	}
}

func (d *DockerClient) findNetwork(name string) (*docker.Network, error) {
	networks, err := d.client.ListNetworks()
	if err != nil {
		return nil, fmt.Errorf("error getting networks: %w", err)
	}

	for _, n := range networks {
		if n.Name == name {
			return &n, nil
		}
	}
	return nil, fmt.Errorf("network '%s' not found", name)
}

func cleanContainerName(name string) string {
	return strings.ReplaceAll(name, "/", "")
}
