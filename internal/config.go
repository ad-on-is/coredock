package internal

import (
	"os"
	"strconv"
	"strings"

	"github.com/thoas/go-funk"
)

type Config struct {
	Domains          []string
	Networks         []string
	TTL              int
	IPPrefixes       []string
	IPPrefixesIgnore []string
}

func NewConfig() *Config {
	c := &Config{
		Domains:    []string{"docker"},
		Networks:   []string{},
		TTL:        300,
		IPPrefixes: []string{},
	}

	domains := os.Getenv("COREDOCK_DOMAINS")
	if domains == "" {
		domains = "docker"
	}
	networks := os.Getenv("COREDOCK_NETWORKS")
	ipPrefixes := os.Getenv("COREDOCK_IP_PREFIXES")
	ipPrefixesIgnore := os.Getenv("COREDOCK_IGNORE_IP_PREFIXES")
	ttlStr := os.Getenv("COREDOCK_TTL")
	ttl := 10

	if t, err := strconv.Atoi(ttlStr); err == nil {
		ttl = t
	}

	c.TTL = ttl
	c.Domains = funk.Filter(strings.Split(domains, ","), func(s string) bool { return s != "" }).([]string)
	c.Domains = funk.Map(c.Domains, func(s string) string { return strings.TrimSpace(s) }).([]string)
	c.Networks = funk.Filter(strings.Split(networks, ","), func(s string) bool { return s != "" }).([]string)
	c.Networks = funk.Map(c.Networks, func(s string) string { return strings.TrimSpace(s) }).([]string)
	c.IPPrefixes = funk.Filter(strings.Split(ipPrefixes, ","), func(s string) bool { return s != "" }).([]string)
	c.IPPrefixes = funk.Map(c.IPPrefixes, func(s string) string { return strings.TrimSpace(s) }).([]string)
	c.IPPrefixesIgnore = funk.Filter(strings.Split(ipPrefixesIgnore, ","), func(s string) bool { return s != "" }).([]string)
	c.IPPrefixesIgnore = funk.Map(c.IPPrefixesIgnore, func(s string) string { return strings.TrimSpace(s) }).([]string)

	return c
}
