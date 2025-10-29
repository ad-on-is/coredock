package internal

import (
	"fmt"
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

func (d *DNSProvider) GetCNAMERecords(service *Service, domain string) []dns.RR {
	rrs := []dns.RR{}

	for _, alias := range service.Aliases {
		rr := new(dns.CNAME)

		ttl := d.config.TTL

		rr.Hdr = dns.RR_Header{
			Name:   fmt.Sprintf("%s.%s.", alias, domain),
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl),
		}
		rr.Target = fmt.Sprintf("%s.%s.", service.Name, domain)

		rrs = append(rrs, rr)
	}

	return rrs
}

func (d *DNSProvider) GetARecords(service *Service, domain string) []dns.RR {
	rrs := []dns.RR{}

	for _, ip := range service.IPs {

		rr := new(dns.A)

		ttl := d.config.TTL

		rr.Hdr = dns.RR_Header{
			Name:   fmt.Sprintf("%s.%s.", service.Name, domain),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl),
		}
		rr.A = ip

		rrs = append(rrs, rr)
	}

	return rrs
}

func (d *DNSProvider) GetPTRRecords(service *Service, domain string) []dns.RR {
	rrs := []dns.RR{}

	for _, ip := range service.IPs {

		rr := new(dns.PTR)

		ttl := d.config.TTL

		ptrName, err := dns.ReverseAddr(ip.String())
		if err != nil {
			continue
		}

		rr.Hdr = dns.RR_Header{
			Name:   ptrName,
			Rrtype: dns.TypePTR,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl),
		}
		rr.Ptr = fmt.Sprintf("%s.%s.", service.Name, domain)

		rrs = append(rrs, rr)
	}

	return rrs
}

func (s *DNSProvider) GetSOARecord(domain string) dns.RR {
	soa := &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   domain + ".",
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    uint32(s.config.TTL),
		},
		Ns:      fmt.Sprintf("coredock.%s.", domain),
		Mbox:    fmt.Sprintf("coredock.coredock.%s.", domain),
		Serial:  uint32(time.Now().Truncate(time.Second).Unix()),
		Refresh: 28800,
		Retry:   7200,
		Expire:  604800,
		Minttl:  uint32(s.config.TTL),
	}
	return soa
}

func (d *DNSProvider) createSRV(prefix string, port int, name string, domain string) dns.RR {
	rr := new(dns.SRV)
	ttl := d.config.TTL

	if prefix == "" {
		prefix = "_http._tcp"
	}

	rr.Hdr = dns.RR_Header{
		Name:   prefix + "." + domain + ".",
		Rrtype: dns.TypeSRV,
		Class:  dns.ClassINET,
		Ttl:    uint32(ttl),
	}

	rr.Port = uint16(port)
	rr.Target = name + "." + domain + "."
	rr.Priority = 10
	rr.Weight = 5

	return rr
}

func (d *DNSProvider) GetSRVRecords(service *Service, domain string) []dns.RR {
	rrs := []dns.RR{}
	if len(service.SRVs) == 0 {
		return rrs
	}

	for _, srv := range service.SRVs {
		rrs = append(rrs, d.createSRV(srv.Prefix, srv.Port, service.Name, domain))
	}

	return rrs
}

func (d *DNSProvider) GetMXRecords(service *Service) []dns.RR {
	rrs := []dns.RR{}

	ttl := d.config.TTL
	for _, n := range service.Hosts {
		rr := new(dns.MX)
		rr.Hdr = dns.RR_Header{
			Name:   n,
			Rrtype: dns.TypeMX,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl),
		}
		rr.Mx = n
		rrs = append(rrs, rr)
	}

	return rrs
}
