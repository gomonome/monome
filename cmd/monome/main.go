package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gomonome/monome"
)

func do(dev monome.Device) {
	dev.Marquee(dev.String(), time.Millisecond*80)
	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 20)
		dev.AllOn()
		time.Sleep(time.Millisecond * 90)
		dev.AllOff()
	}
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

	sigchan := make(chan os.Signal, 10)

	// listen for ctrl+c
	go signal.Notify(sigchan, os.Interrupt)

	// interrupt has happend
	<-sigchan

	fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")

	for _, dev := range devices {
		dev.StopListening()
		time.Sleep(time.Millisecond * 100)
		dev.Close()
	}
	fmt.Fprintln(os.Stdout, "done")
	os.Exit(0)
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
