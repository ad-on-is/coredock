package internal

import (
	"os"
	"strings"

	"github.com/alecthomas/kingpin"
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

	app := kingpin.New("coredock", "CoreDNS Docker integration")
	var (
		domains    = app.Flag("domains", "Domains to use").String()
		networks   = app.Flag("networks", "Auto-connect containers to these networks").String()
		ttl        = app.Flag("ttl", "Time to live for DNS records.").Default("300").Int()
		ipPrefixes = app.Flag("ip-prefixes", "Only include IPs with the given prefix.").String()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	c.TTL = *ttl
	c.Domains = strings.Split(*domains, ",")
	c.Networks = strings.Split(*networks, ",")
	c.IPPrefixes = strings.Split(*ipPrefixes, ",")

	return c
}
