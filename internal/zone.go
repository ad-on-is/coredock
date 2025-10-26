package internal

import (
	"fmt"
	"os"
	"regexp"
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

func (z *ZoneHandler) getZoneFile(domain string) (string, error) {
	f := "/tmp/coredock/db." + domain
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		if err := os.WriteFile(f, []byte(""), 0o644); err != nil {
			return "", fmt.Errorf("error creating zone file for domain %s: %s", domain, err)
		}
		return "", nil
	}
	data, err := os.ReadFile("/tmp/coredock/db." + domain)
	if err != nil {
		return "", fmt.Errorf("error reading zone file for domain %s: %s", domain, err)
	}
	return string(data), nil
}

func replaceWrapper(contents string, str string, start string, end string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)%s.*?%s`,
		regexp.QuoteMeta(start),
		regexp.QuoteMeta(end)))

	if !pattern.MatchString(contents) {
		contents = fmt.Sprintf("%s\n%s\n%s\n%s", contents, start, str, end)
	} else {
		contents = pattern.ReplaceAllStringFunc(contents, func(match string) string {
			return fmt.Sprintf("%s\n%s\n%s", start, str, end)
		})
	}

	return contents
}

func (z *ZoneHandler) writeZoneEntry(domain string, name string, delete bool, soa dns.RR, recordset ...[]dns.RR) error {
	z.mux.Lock()
	defer z.mux.Unlock()
	contents, err := z.getZoneFile(domain)
	if err != nil {
		return err
	}

	records := ""

	if delete {
		contents = replaceWrapper(contents, "", fmt.Sprintf("; %s BEGIN", name), fmt.Sprintf("; %s END", name))
	} else {
		header := fmt.Sprintf("$ORIGIN %s.\n$TTL %d\n%s\n", domain, z.config.TTL, soa.String())
		for _, set := range recordset {
			for _, r := range set {
				records += r.String() + "\n"
			}
		}
		contents = replaceWrapper(contents, header, "; header BEGIN", "; header END")
		contents = replaceWrapper(contents, records, fmt.Sprintf("; %s BEGIN", name), fmt.Sprintf("; %s END", name))
	}

	err = os.WriteFile("/tmp/coredock/db."+domain, []byte(contents), 0o644)
	if err != nil {
		return fmt.Errorf("error writing zone file: %s", err)
	}
	return nil
}

func (z *ZoneHandler) Create(s *Service, d *DNSProvider) {
	logger.Infof("Service %sed: %s", s.Action, s.Name)
	for _, domain := range s.Domains {

		soa := d.GetSOARecord(s, domain)

		aRecords := d.GetARecords(s, domain)

		srvRecords := d.GetSRVRecords(s, domain)

		z.writeZoneEntry(domain, s.Name, false, soa, aRecords, srvRecords)

	}
}

func (z *ZoneHandler) Delete(s *Service) {
	logger.Infof("Service %sed: %s", s.Action, s.Name)

	for _, domain := range s.Domains {
		z.writeZoneEntry(domain, s.Name, true, nil, []dns.RR{})
	}
}
