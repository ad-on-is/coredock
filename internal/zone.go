package internal

import (
	"fmt"
	"os"
	"strings"
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

func (z *ZoneHandler) createZoneFile(zone string) error {
	f := "/tmp/coredock/db." + zone
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		if err := os.WriteFile(f, []byte(""), 0o644); err != nil {
			return fmt.Errorf("error creating zone file for zone %s: %s", zone, err)
		}
	}
	return nil
}

func (z *ZoneHandler) writeZoneEntry(zone string, soa dns.RR, records []dns.RR) error {
	z.mux.Lock()
	defer z.mux.Unlock()
	err := z.createZoneFile(zone)
	if err != nil {
		return err
	}

	contents := fmt.Sprintf("$ORIGIN %s.\n$TTL %d\n%s\n", zone, z.config.TTL, soa.String())

	for _, r := range records {
		contents += r.String() + "\n"
	}

	err = os.WriteFile("/tmp/coredock/db."+zone, []byte(contents), 0o644)
	if err != nil {
		return fmt.Errorf("error writing zone file: %s", err)
	}
	return nil
}

func (z *ZoneHandler) Update(services *[]Service, d *DNSProvider) {
	soas := map[string]dns.RR{}
	records := map[string][]dns.RR{}
	reverseRecords := map[string][]dns.RR{}
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
			records[domain] = append(records[domain], d.GetCNAMERecords(&s, domain)...)
			records[domain] = append(records[domain], d.GetSRVRecords(&s, domain)...)

			for _, ip := range s.IPs {
				ipstr := ip.String()
				split := strings.Split(ipstr, ".")
				zone1 := fmt.Sprintf("%s.in-addr.arpa", split[0])
				zone2 := fmt.Sprintf("%s.%s.in-addr.arpa", split[1], split[0])
				zone3 := fmt.Sprintf("%s.%s.%s.in-addr.arpa", split[2], split[1], split[0])
				if _, ok := reverseRecords[zone1]; !ok {
					reverseRecords[zone1] = []dns.RR{}
				}
				if _, ok := reverseRecords[zone2]; !ok {
					reverseRecords[zone2] = []dns.RR{}
				}

				if _, ok := reverseRecords[zone3]; !ok {
					reverseRecords[zone3] = []dns.RR{}
				}
				reverseRecords[zone1] = append(reverseRecords[zone1], d.GetPTRRecords(&s, domain)...)
				reverseRecords[zone2] = append(reverseRecords[zone2], d.GetPTRRecords(&s, domain)...)
				reverseRecords[zone3] = append(reverseRecords[zone3], d.GetPTRRecords(&s, domain)...)

			}
		}
		logger.Infof("Service '%s' added with IPs: %s", s.Name, s.IPs)
	}
	for domain, soa := range soas {
		if err := z.writeZoneEntry(domain, soa, records[domain]); err != nil {
			logger.Errorf("Error writing zone entry for domain %s: %s", domain, err)
		}
	}
	for zone, rrs := range reverseRecords {
		if err := z.writeZoneEntry(zone, d.GetSOARecord(zone), rrs); err != nil {
			logger.Errorf("Error writing reverse zone entry for zone %s: %s", zone, err)
		}
	}
}
