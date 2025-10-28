package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff/v5"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/thoas/go-funk"
)

type DockerClient struct {
	client  *docker.Client
	channel chan *Service
	config  *Config
}

func NewDockerClient(channel chan *Service, conf *Config) (*DockerClient, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: client, channel: channel, config: conf}, nil
}

func (d *DockerClient) Run() error {
	dockerChan := make(chan *docker.APIEvents)

	d.client.AddEventListener(dockerChan)

	containers, err := d.getContainers()
	if err != nil {
		return fmt.Errorf("error getting containers: %w", err)
	}

	go func() {
		for _, c := range containers {
			d.maybeConnectToNetwork(&c)
			d.channel <- NewService(&c, "start", d.config)
		}
	}()

	for e := range dockerChan {
		logger.Debugf("Received event from Docker: %v", e)
		d.handleService(e)
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
		_, ok := labels["coredock.ignore"]
		if ok {
			logger.Debugf("Ignoring container '%s' due to 'coredock.ignore' label", cleanContainerName(c.Names[0]))
		}
		return !ok
	}).([]docker.APIContainers), nil
}

func (d *DockerClient) findContainer(id string) (*docker.APIContainers, error) {
	return backoff.Retry(context.TODO(), func() (*docker.APIContainers, error) {
		containers, err := d.getContainers()
		if err != nil {
			return nil, fmt.Errorf("error getting containers: %w", err)
		}

		for _, c := range containers {
			ips := 0
			if c.ID == id {
				for _, nw := range c.Networks.Networks {
					for range nw.IPAddress {
						ips++
					}
				}
				if ips == 0 {
					return nil, fmt.Errorf("container '%s' has no IP address assigned yet", shortContainerID(id))
				}
				return &c, nil
			}
		}
		return nil, fmt.Errorf("container '%s' not found", shortContainerID(id))
	}, backoff.WithMaxTries(5), backoff.WithBackOff(backoff.NewExponentialBackOff()))
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
		logger.Debugf("Could not handle service for action '%s': %v", e.Action, err)
		return
	}

	d.maybeConnectToNetwork(c)
	d.channel <- NewService(c, e.Action, d.config)
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

func shortContainerID(id string) string {
	if len(id) < 10 {
		return id
	}
	return id[:10]
}
