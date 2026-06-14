package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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
	db            *DB
	config        *Config
	previousNames []string
	mux           sync.Mutex
}

func NewDockerClient(channel chan *[]Service, conf *Config, db *DB) (*DockerClient, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return &DockerClient{client: client, channel: channel, config: conf, db: db, previousNames: []string{}, mux: sync.Mutex{}}, nil
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

	currentNames := funk.Map(services, func(s Service) string {
		sort.Slice(s.IPs, func(i, j int) bool {
			return bytes.Compare(s.IPs[i], s.IPs[j]) < 0
		})
		return fmt.Sprintf("%s%v", s.Name, s.IPs)
	}).([]string)

	sort.Strings(currentNames)

	pc, cc := funk.DifferenceString(d.previousNames, currentNames)

	if len(pc) > 0 {
		logger.Infof("Detected changed containers: %v", pc)
	}
	if len(cc) > 0 {
		logger.Infof("Detected changed containers: %v", cc)
	}

	if len(pc) > 0 || len(cc) > 0 {
		d.previousNames = currentNames
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

func (d *DockerClient) connectWithPriority(networkID, containerID string, priority int, containerIP string) error {

	pl := map[string]any{
		"Container": containerID,
		"EndpointConfig": map[string]any{
			"GwPriority": priority,
			"IPAMConfig": map[string]any{
				"IPv4Address": containerIP,
			},
		},
	}

	if containerIP != "" {
		delete(pl["EndpointConfig"].(map[string]any), "IPAMConfig")
	}

	payload, _ := json.Marshal(pl)

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

	resp, err := httpc.Post(
		fmt.Sprintf("http://localhost/networks/%s/connect", networkID),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		// already connected
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker API error: %s", string(body))
	}
	return nil
}

func (d *DockerClient) isIPAssignedOnNetwork(networkName, ip string) (bool, string) {
	containers, err := d.client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return false, ""
	}
	for _, c := range containers {
		for nwName, nw := range c.Networks.Networks {
			if nwName == networkName && nw.IPAddress == ip {
				return true, cleanContainerName(c.Names[0])
			}
		}
	}
	return false, ""
}

func (d *DockerClient) maybeConnectToNetwork(c *docker.APIContainers) {
	containerName := cleanContainerName(c.Names[0])
	for _, nw := range d.config.Networks {

		dnw, err := d.findNetwork(nw)
		if err != nil {
			logger.Errorf("Error finding network '%s': %v", nw, err)
			continue
		}

		dbKey := fmt.Sprintf("%s-%s", containerName, dnw.Name)
		containerIp := ""

		if d.config.ReuseIPs {

			containerIp = d.db.Get(dbKey)
			logger.Debugf("container ip: %s", containerIp)
			assigned, assignedTo := d.isIPAssignedOnNetwork(dnw.Name, containerIp)
			if containerIp != "" {
				if assigned && containerName != assignedTo {
					containerIp = ""
				} else {
					logger.Infof("Found saved IP '%s' for '%s'", containerIp, containerName)
				}
			}
		}

		logger.Debugf("Connected '%s' to network '%s'", containerName, dnw.Name)
		if dnw.Driver == "macvlan" {
			err = d.connectWithPriority(dnw.ID, c.ID, 9999, containerIp)
		} else {
			opts := docker.NetworkConnectionOptions{
				Container: c.ID,
			}
			if containerIp != "" {
				opts.EndpointConfig = &docker.EndpointConfig{

					IPAMConfig: &docker.EndpointIPAMConfig{
						IPv4Address: containerIp,
					},
				}
			}
			err = d.client.ConnectNetwork(dnw.ID, opts)

		}

		if err != nil && !strings.Contains(err.Error(), "already exists") {
			logger.Errorf("Error connecting container '%s' to network '%s': %v", cleanContainerName(c.Names[0]), dnw.Name, err)
		} else {
			d.saveIp(c, dnw)
		}
	}

}

func (d *DockerClient) saveIp(c *docker.APIContainers, dnw *docker.Network) {
	containerName := cleanContainerName(c.Names[0])
	dbKey := fmt.Sprintf("%s-%s", containerName, dnw.Name)
	inspected, err := d.client.InspectContainerWithOptions(docker.InspectContainerOptions{
		ID: c.ID,
	})

	if err != nil {
		logger.Errorf("Error inspecting container '%s': %v", containerName, err)
		return
	}

	for name, ep := range inspected.NetworkSettings.Networks {
		if name == dnw.Name {
			ip := ep.IPAddress
			d.db.Set(dbKey, ip)
		}
	}
}

func (d *DockerClient) getContainerIPs(c *docker.APIContainers, network string) net.IP {
	for _, netw := range c.Networks.Networks {
		if network != "" && netw.NetworkID != network {
			continue
		}
		ip := net.ParseIP(netw.IPAddress)
		if ip != nil {
			return ip
		}
	}
	return nil
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
