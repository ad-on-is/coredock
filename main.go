package main

import (
	"github.com/ad-on-is/coredock/internal"
	"github.com/thoas/go-funk"
)

var logger = internal.InitLogger(0)

func main() {
	err := internal.CreateZoneDir()
	if err != nil {
		logger.Errorf("Error initializing zone files: %s", err)
		panic(1)
	}
	internal.InitLogger(0)
	config := internal.NewConfig()

	serviceChan := make(chan *internal.Service)
	d, err := internal.NewDockerClient(serviceChan, config)
	if err != nil {
		panic(err)
	}
	zone := internal.NewZoneHandler(config)
	dns := internal.NewDNSProvider(config)
	go func() {
		d.Run()
	}()

	for s := range serviceChan {
		createHandler := []string{"create", "start", "connect"}

		if funk.Contains(createHandler, s.Action) {
			zone.Create(s, dns)
		} else {
			zone.Delete(s)
		}

		// deleteZone

	}
}
