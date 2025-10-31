package main

import (
	"github.com/ad-on-is/coredock/internal"
)

var (
	logger  = internal.InitLogger()
	Version string
)

func main() {
	logger.Info(`
=================================
                   _         _   
 ___ ___ ___ ___ _| |___ ___| |_ 
|  _| . |  _| -_| . | . |  _| '_|
|___|___|_| |___|___|___|___|_,_|
                                
Expose your Docker containers via DNS.
version: ` + Version + `
=================================
		`)
	err := internal.CreateZoneDir()
	if err != nil {
		logger.Errorf("Error initializing zone files: %s", err)
		panic(1)
	}
	config := internal.NewConfig()

	internal.InitLogger()
	serviceChan := make(chan *[]internal.Service)
	d, err := internal.NewDockerClient(serviceChan, config)
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
