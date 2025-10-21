package main

import (
	"fmt"

	"github.com/ad-on-is/coredock/internal"
)

func main() {
	containerChan := make(chan internal.Container)
	d, err := internal.NewDockerClient(containerChan)
	if err != nil {
		panic(err)
	}

	go func() {
		d.Run()
	}()

	for c := range containerChan {
		writeZoneFile(&c)
	}
}

func writeZoneFile(c *internal.Container) {
	fmt.Println("COntainer here", c.Name)
}
