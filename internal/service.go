package internal

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/thoas/go-funk"
)

type SRV struct {
	Prefix string
	Name   string
	Port   int
}

type Service struct {
	ID      string
	Name    string
	IPs     []net.IP
	Aliases []string
	Domains []string
	Hosts   []string
	Action  string
	Ignore  bool
	SRVs    []SRV
}

func (s Service) String() string {
	jsonStr, _ := json.MarshalIndent(s, "", "  ")
	return string(jsonStr)
}

func NewService(c *docker.APIContainers, action string, conf *Config) *Service {
	ips := []net.IP{}
	for _, netw := range c.Networks.Networks {
		ip := net.ParseIP(netw.IPAddress)
		if ip != nil {

			foundPrefix := false
			ignorePrefix := false

			for _, p := range conf.IPPrefixes {
				if strings.HasPrefix(ip.String(), p) {
					foundPrefix = true
					break
				}
			}

			for _, p := range conf.IPPrefixesIgnore {
				if strings.HasPrefix(ip.String(), p) {

					ignorePrefix = true
					break
				}
			}
			if !foundPrefix && len(conf.IPPrefixes) > 0 || ignorePrefix && len(conf.IPPrefixesIgnore) > 0 {
				continue
			}

			ips = append(ips, ip)
		}
	}

	s := &Service{
		ID:      c.ID,
		Action:  action,
		IPs:     ips,
		Aliases: []string{},
		Ignore:  false,
		SRVs:    []SRV{},
		Name:    cleanContainerName(c.Names[0]),
	}
	s = s.ParseLabels(c)

	s.Aliases = funk.UniqString(s.Aliases)

	s.Domains = append(s.Domains, conf.Domains...)
	s.Domains = funk.UniqString(s.Domains)

	for _, d := range s.Domains {
		s.Hosts = append(s.Hosts, fmt.Sprintf("%s.%s", s.Name, d))
		for _, a := range s.Aliases {
			s.Hosts = append(s.Hosts, fmt.Sprintf("%s.%s", a, d))
		}
	}

	s.Hosts = funk.UniqString(s.Hosts)

	logger.Infof("%v", s.Aliases)

	return s
}

func (s *Service) GetHosts(domain string) []string {
	return funk.FilterString(s.Hosts, func(h string) bool {
		return strings.HasSuffix(h, domain)
	})
}

func (s *Service) ParseLabels(c *docker.APIContainers) *Service {
	for key, value := range c.Labels {
		if !strings.HasPrefix(key, "coredock") {
			continue
		}
		// logger.Debugf("%s: %v", key, value)
		if key == "coredock.ignore" {
			s.Ignore = true
		}
		if key == "coredock.domains" {
			ds := funk.Map(strings.Split(value, ","), func(d string) string {
				return strings.TrimSpace(d)
			}).([]string)
			s.Domains = append(s.Domains, ds...)
		}

		if key == "coredock.alias" {
			as := funk.Map(strings.Split(value, ","), func(a string) string {
				return strings.TrimSpace(a)
			}).([]string)
			s.Aliases = append(s.Aliases, as...)
		}

		if strings.HasPrefix(key, "coredock.srv") {
			split := strings.Split(key, "#")
			port, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			srv := SRV{Port: port}
			if len(split) == 1 {
				srv.Prefix = fmt.Sprintf("_http._tcp.%s", s.Name)
				srv.Name = s.Name
			}
			if len(split) == 2 {
				name := split[1]
				pattern := "_([a-zA-Z0-9]+)._([a-zA-Z0-9]+).(.*)"
				re := regexp.MustCompile(pattern)
				matches := re.FindStringSubmatch(name)
				if len(matches) == 4 {
					name = matches[3]
					srv.Prefix = fmt.Sprintf("_%s._%s.%s", matches[1], matches[2], name)
					srv.Name = matches[3]
				} else {
					srv.Prefix = fmt.Sprintf("_http._tcp.%s", name)
					srv.Name = name
				}
			}
			s.SRVs = append(s.SRVs, srv)
			if srv.Name != s.Name {
				s.Aliases = append(s.Aliases, srv.Name)
			}
		}
	}

	return s
}
