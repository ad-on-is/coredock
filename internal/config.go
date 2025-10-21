package internal

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Domains    []string
	Networks   []string
	TTL        int
	IPPrefixes []string
}

func NewConfig() *Config {
	c := &Config{
		Domains:    []string{"docker"},
		Networks:   []string{},
		TTL:        300,
		IPPrefixes: []string{},
	}

	domains := os.Getenv("COREDOCK_DOMAINS")
	networks := os.Getenv("COREDOCK_NETWORKS")
	ipPrefixes := os.Getenv("COREDOCK_IP_PREFIXES")
	ttlStr := os.Getenv("COREDOCK_TTL")
	ttl := 300

	if t, err := strconv.Atoi(ttlStr); err == nil {
		ttl = t
	}

	c.TTL = ttl
	c.Domains = strings.Split(domains, ",")
	c.Networks = strings.Split(networks, ",")
	c.IPPrefixes = strings.Split(ipPrefixes, ",")
	return c
}
