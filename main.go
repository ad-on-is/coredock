package main

import (
	"github.com/ad-on-is/coredock/internal"
	"github.com/thoas/go-funk"
)

var logger = internal.InitLogger()

func main() {
	err := internal.CreateZoneDir()
	if err != nil {
		logger.Errorf("Error initializing zone files: %s", err)
		panic(1)
	}
	config := internal.NewConfig()

	internal.InitLogger()
	serviceChan := make(chan *internal.Service)
	d, err := internal.NewDockerClient(serviceChan, config)
	if err != nil {
		panic(err)
	}
	zone := internal.NewZoneHandler(config)
	dns := internal.NewDNSProvider(config)

	go func() {
		err := d.Run()
		if err != nil {
			panic(err)
		}
	}()

	for s := range serviceChan {
		createHandler := []string{"create", "start", "connect"}

		logger.Debugf("Handling container '%s': %s", s.Name, s.Action)

		if funk.Contains(createHandler, s.Action) {
			zone.Create(s, dns)
		} else {
			zone.Delete(s)
		}

	}
}
