package main

import (
	"github.com/ad-on-is/coredock/internal"
)

var (
	logger  = internal.InitLogger()
	Version string
)

func main() {
	config := internal.NewConfig()

	logger.Infof(`
=================================
                   _         _
 ___ ___ ___ ___ _| |___ ___| |_
|  _| . |  _| -_| . | . |  _| '_|
|___|___|_| |___|___|___|___|_,_|

Expose your Docker containers via DNS.
Version: %s

Domains: %v
IP-Prefixes: %v
Networks: %v
=================================
		`, Version, config.Domains, config.IPPrefixes, config.Networks)
	err := internal.CreateZoneDir()
	if err != nil {
		logger.Errorf("Error initializing zone files: %s", err)
		panic(1)
	}

	internal.InitLogger()
	serviceChan := make(chan *[]internal.Service)
	db := internal.NewDB()
	d, err := internal.NewDockerClient(serviceChan, config, db)
	if err != nil {
		panic(err)
	}
	zone := internal.NewZoneHandler(config)
	dns := internal.NewDNSProvider(config)

	go func() {
		err := d.Run()
		if err != nil {
			logger.Errorf("Error running Docker client: %s", err)
			panic(1)
		}
	}()

	for s := range serviceChan {
		zone.Update(s, dns)
	}
}
