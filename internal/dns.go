package internal

import (
	"strings"
	"time"

	"github.com/miekg/dns"
)

type DNSProvider struct {
	config *Config
}

func NewDNSProvider(c *Config) *DNSProvider {
	return &DNSProvider{
		config: c,
	}
}

func (d *DNSProvider) GetARecords(service *Service, domain string) []dns.RR {
	rrs := []dns.RR{}

	for _, n := range service.GetHosts(domain) {
		for _, ip := range service.IPs {

			rr := new(dns.A)

			ttl := d.config.TTL

			rr.Hdr = dns.RR_Header{
				Name:   n + ".",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl),
			}
			rr.A = ip

			rrs = append(rrs, rr)
		}
	}

	return rrs
}

func (s *DNSProvider) GetSOARecord(service *Service, domain string) dns.RR {
	dom := dns.Fqdn(domain + ".")
	soa := &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   dom,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    uint32(s.config.TTL),
		},
		Ns:      "coredock." + dom,
		Mbox:    "coredock.coredock." + dom,
		Serial:  uint32(time.Now().Truncate(time.Second).Unix()),
		Refresh: 28800,
		Retry:   7200,
		Expire:  604800,
		Minttl:  uint32(s.config.TTL),
	}
	return soa
}

func (d *DNSProvider) createSRV(port int, name string) dns.RR {
	rr := new(dns.SRV)
	ttl := d.config.TTL

	rr.Hdr = dns.RR_Header{
		Name:   name + ".",
		Rrtype: dns.TypeSRV,
		Class:  dns.ClassINET,
		Ttl:    uint32(ttl),
	}

	rr.Port = uint16(port)
	rr.Target = name
	rr.Priority = 10
	rr.Weight = 5

	return rr
}

func (d *DNSProvider) GetSRVRecords(service *Service, domain string) []dns.RR {
	if len(service.SRVs) == 0 {
		return []dns.RR{}
	}

	rrs := make([]dns.RR, 0)

	for _, port := range service.SRVs {
		for _, h := range service.GetHosts(domain) {
			rrs = append(rrs, d.createSRV(port, h))
		}
	}

	return rrs
}

// func GetMXRecord(n string, service *Service) dns.RR {
// 	rr := new(dns.MX)
//
// 	var ttl int
// 	if service.TTL != -1 {
// 		ttl = service.TTL
// 	} else {
// 		ttl = d.config.TTL
// 	}
//
// 	rr.Hdr = dns.RR_Header{
// 		Name:   n,
// 		Rrtype: dns.TypeMX,
// 		Class:  dns.ClassINET,
// 		TTL:    uint32(ttl),
// 	}
//
// 	rr.Mx = n
//
// 	return rr
// }

func askerInSameNet(asker string, ip string) int {
	if asker == "127.0.0.1" {
		return 1
	}
	a := strings.Split(asker, ".")
	i := strings.Split(ip, ".")

	if a[0] == i[0] && a[1] == i[1] && a[2] == i[2] {
		return 3
	}

	if a[0] == i[0] && a[1] == i[1] {
		return 2
	}

	if a[0] == i[0] {
		return 1
	}

	return 0
}
