package main

import (
	"fmt"
	"os"

	"github.com/gomonome/monome"
)

func do(dev monome.Device) {
	dev.Print(dev.String())
	dev.SetHandler(monome.HandlerFunc(Handle))
	dev.StartListening(func(err error) {
		fmt.Fprintf(os.Stderr, "can't read from monome %s: %v\n", dev, err)
	})
}

func main() {
	if os.Getenv("USER") != "root" {
		fmt.Fprintln(os.Stderr, "please run as root")
		os.Exit(1)
	}
	var c chan bool

	devices, err := monome.Devices()

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v", err)
		os.Exit(1)
	}

	if len(devices) < 1 {
		fmt.Fprintln(os.Stderr, "no monome device found")
		os.Exit(1)
	}

	for _, dev := range devices {
		go func(d monome.Device) {
			do(d)
		}(dev)
	}

	<-c
	for _, dev := range devices {
		dev.StopListening()
		dev.Close()
	}
}

// highlight the pressed buttons
func Handle(d monome.Device, x, y uint8, down bool) {
	if down {
		fmt.Printf("%s pressed %v/%v\n", d, x, y)
		d.On(x, y)
		return
	}
	fmt.Printf("%s released %v/%v\n", d, x, y)
	d.Off(x, y)
}
