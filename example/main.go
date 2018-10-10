package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gomonome/monome"
)

func setup(dev monome.Device) {
	dev.Marquee(dev.String(), time.Millisecond*100)
	dev.SwitchAll(true)
	time.Sleep(time.Microsecond * 100)
	dev.SwitchAll(false)
	dev.SetHandler(monome.HandlerFunc(func(d monome.Device, x, y uint8, down bool) {
		action := "released"
		if down {
			action = "pressed"
		}
		fmt.Printf("%s %s key %v/%v\n", d, action, x, y)

		// switch the lights
		d.Switch(x, y, down)
	}))
	dev.StartListening(func(err error) {
		// aborting on io error
		sigchan <- os.Interrupt
	})
}

var sigchan = make(chan os.Signal, 10)

func main() {
	if os.Getenv("USER") != "root" {
		fmt.Fprintln(os.Stderr, "please run as root")
		os.Exit(1)
	}

	devices, _ := monome.Devices()

	if len(devices) < 1 {
		fmt.Fprintln(os.Stderr, "no monome device found")
		os.Exit(1)
	}

	for _, dev := range devices {
		setup(dev)
	}

	// listen for ctrl+c
	go signal.Notify(sigchan, os.Interrupt)

	// interrupt has happend
	<-sigchan

	fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")

	for _, dev := range devices {
		dev.Close()
	}

	fmt.Fprintln(os.Stdout, "done")
	os.Exit(0)
}
