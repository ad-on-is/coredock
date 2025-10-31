package internal

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/thoas/go-funk"
)

type DockerClient struct {
	client        *docker.Client
	channel       chan *[]Service
	config        *Config
	previousNames []string
	currentNames  []string
	mux           sync.Mutex
}

func NewDockerClient(channel chan *[]Service, conf *Config) (*DockerClient, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: client, channel: channel, config: conf, previousNames: []string{}, currentNames: []string{}, mux: sync.Mutex{}}, nil
}

func (d *DockerClient) sendContainers() {
	d.mux.Lock()
	containers, err := d.getContainers()
	if err != nil {
		return
	}

	services := []Service{}
	for _, c := range containers {
		d.maybeConnectToNetwork(&c)
		services = append(services, *NewService(&c, "start", d.config))
	}

	d.currentNames = funk.Map(services, func(s Service) string {
		sort.Slice(s.IPs, func(i, j int) bool {
			return bytes.Compare(s.IPs[i], s.IPs[j]) < 0
		})
		return fmt.Sprintf("%s%v", s.Name, s.IPs)
	}).([]string)

	sort.Strings(d.currentNames)

	pc, cc := funk.DifferenceString(d.previousNames, d.currentNames)

	if len(pc) > 0 || len(cc) > 0 {
		d.previousNames = d.currentNames
		d.channel <- &services
	}
	d.mux.Unlock()
}

func debounce(fn func(), delay time.Duration) func() {
	var timer *time.Timer
	var mu sync.Mutex

	return func() {
		mu.Lock()
		defer mu.Unlock()

		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(delay, fn)
	}
}

func (d *DockerClient) Run() error {
	dockerChan := make(chan *docker.APIEvents)

	d.sendContainers()

	go func() {
		for {
			time.Sleep(10 * time.Second)
			d.sendContainers()
		}
	}()

	err := d.client.AddEventListener(dockerChan)
	if err != nil {
		return fmt.Errorf("error adding Docker event listener: %w", err)
	}

	for e := range dockerChan {

		logger.Debugf("Received event from Docker: %v", e)
		actions := []string{"create", "connect", "disconnect", "destroy", "start", "stop"}

		if funk.Contains(actions, e.Action) {
			debounce(d.sendContainers, 5*time.Second)()
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

		isRunning := c.State == "running"

		return !isIgnored && !isCoredock && isRunning
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
