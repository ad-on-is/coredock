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
			panic(err)
		}
	}()

	previousNames := []string{}

	for s := range serviceChan {
		currentNames := funk.Map(*s, func(serv internal.Service) string {
			return serv.Name
		}).([]string)

		pc, cc := funk.DifferenceString(previousNames, currentNames)
		if len(pc) > 0 || len(cc) > 0 {
			zone.Update(s, dns)
			previousNames = currentNames
		}

	}
}
