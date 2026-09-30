package internal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sort"
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
		ipv6 := net.ParseIP(netw.GlobalIPv6Address)
		if ip == nil && ipv6 == nil {
			continue
		}

		foundPrefix := false
		ignorePrefix := false

		for _, p := range conf.IPPrefixes {
			if strings.HasPrefix(ip.String(), p) || ipv6 != nil && strings.HasPrefix(ipv6.String(), p) {
				foundPrefix = true
				break
			}
		}

		for _, p := range conf.IPPrefixesIgnore {
			if strings.HasPrefix(ip.String(), p) || ipv6 != nil && strings.HasPrefix(ipv6.String(), p) {

				ignorePrefix = true
				break
			}
		}
		if !foundPrefix && len(conf.IPPrefixes) > 0 || ignorePrefix && len(conf.IPPrefixesIgnore) > 0 {
			continue
		}

		ips = append(ips, ip)
		if ipv6 != nil {
			ips = append(ips, ipv6)
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

	return s
}

func (s *Service) Hash() string {
	b, _ := json.Marshal(s)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) GetHosts(domain string) []string {
	return funk.FilterString(s.Hosts, func(h string) bool {
		return strings.HasSuffix(h, domain)
	})
}

func injectLabels(s *Service, labels map[string]string, label string) {

	switch label {
	case "ignore":
		for key, _ := range labels {
			if key == "coredock.ignore" {
				s.Ignore = true
				return
			}
		}
		return
	case "domains":
		for key, value := range labels {
			if key != "coredock.domains" {
				continue
			}
			ds := funk.Map(strings.Split(value, ","), func(d string) string {
				return strings.TrimSpace(d)
			}).([]string)
			s.Domains = append(s.Domains, ds...)
		}
	case "aliases":
		for key, value := range labels {
			if key != "coredock.aliases" {
				continue
			}
			as := funk.Map(strings.Split(value, ","), func(a string) string {
				return strings.TrimSpace(a)
			}).([]string)
			s.Aliases = append(s.Aliases, as...)
		}
	case "srv":

		for key, value := range labels {
			if key != "coredock.srv" {
				continue
			}
			port, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			s.SRVs = append(s.SRVs, SRV{Port: port, Name: s.Name, Prefix: fmt.Sprintf("_http._tcp.%s", s.Name)})

			for _, a := range s.Aliases {
				s.SRVs = append(s.SRVs, SRV{Name: a, Prefix: fmt.Sprintf("_http._tcp.%s", a), Port: port})
			}
		}
	case "srvs":
		for key, value := range labels {
			if !strings.HasPrefix(key, "coredock.srv--") {
				continue
			}
			split := strings.Split(key, "--")
			port, err := strconv.Atoi(value)
			if err != nil {
				continue
			}

			if len(split) != 2 {
				return
			}
			srv := SRV{Port: port}
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
			if srv.Name != s.Name {
				s.Aliases = append(s.Aliases, srv.Name)
			}

			s.SRVs = append(s.SRVs, srv)

		}
	default:
		return
	}

}

func (s *Service) ParseLabels(c *docker.APIContainers) *Service {
	injectLabels(s, c.Labels, "ignore")
	if s.Ignore {
		return s
	}
	injectLabels(s, c.Labels, "domains")
	injectLabels(s, c.Labels, "aliases")
	injectLabels(s, c.Labels, "srv")
	injectLabels(s, c.Labels, "srvs")
	sort.Strings(s.Aliases)
	sort.Strings(s.Hosts)
	sort.Strings(s.Domains)
	sort.Slice(s.IPs, func(i, j int) bool {
		return bytes.Compare(s.IPs[i], s.IPs[j]) < 0
	})
	sort.Slice(s.SRVs, func(i, j int) bool {
		return bytes.Compare([]byte(s.SRVs[i].Name), []byte(s.SRVs[j].Name)) < 0
	})

	return s
}
