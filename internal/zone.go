package internal

import (
	"fmt"
	"os"
	"sync"

	"github.com/miekg/dns"
)

type ZoneHandler struct {
	config *Config
	mux    *sync.Mutex
}

func NewZoneHandler(config *Config) *ZoneHandler {
	return &ZoneHandler{config: config, mux: &sync.Mutex{}}
}

func CreateZoneDir() error {
	err := os.MkdirAll("/tmp/coredock", 0o755)
	if err != nil {
		return fmt.Errorf("error creating /tmp/coredock directory: %s", err)
	}

	return nil

	// Placeholder function to match the file name.
}

func (z *ZoneHandler) createZoneFile(domain string) error {
	f := "/tmp/coredock/db." + domain
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		if err := os.WriteFile(f, []byte(""), 0o644); err != nil {
			return fmt.Errorf("error creating zone file for domain %s: %s", domain, err)
		}
	}
	return nil
}

func (z *ZoneHandler) writeZoneEntry(domain string, soa dns.RR, records []dns.RR) error {
	z.mux.Lock()
	defer z.mux.Unlock()
	err := z.createZoneFile(domain)
	if err != nil {
		return err
	}

	contents := fmt.Sprintf("$ORIGIN %s.\n$TTL %d\n%s\n", domain, z.config.TTL, soa.String())

	for _, r := range records {
		contents += r.String() + "\n"
	}

	err = os.WriteFile("/tmp/coredock/db."+domain, []byte(contents), 0o644)
	if err != nil {
		return fmt.Errorf("error writing zone file: %s", err)
	}
	return nil
}

func (z *ZoneHandler) Update(services *[]Service, d *DNSProvider) {
	soas := map[string]dns.RR{}
	records := map[string][]dns.RR{}
	for _, s := range *services {

		if len(s.IPs) == 0 {
			logger.Warnf("Service '%s' skipped: No valid IP address found.", s.Name)
			continue
		}
		for _, domain := range s.Domains {

			if _, ok := soas[domain]; !ok {
				soas[domain] = d.GetSOARecord(domain)
			}

			if _, ok := records[domain]; !ok {
				records[domain] = []dns.RR{}
			}
			records[domain] = append(records[domain], d.GetARecords(&s, domain)...)

			records[domain] = append(records[domain], d.GetSRVRecords(&s, domain)...)

		}
		logger.Infof("Service '%s' added with IPs: %s", s.Name, s.IPs)
	}
	for domain, soa := range soas {
		z.writeZoneEntry(domain, soa, records[domain])
	}
}
