package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gomonome/monome"
)

func do(dev monome.Device) error {
	err := dev.Marquee(dev.String(), time.Millisecond*80)
	if err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 20)
		err = dev.SwitchAll(true)
		if err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 90)
		err = dev.SwitchAll(false)
		if err != nil {
			return err
		}
	}
	dev.SetHandler(monome.HandlerFunc(Handle))
	dev.StartListening(func(err error) {
		fmt.Fprintf(os.Stderr, "can't read from monome %s: %v, stop listening\n", dev, err)
		dev.StopListening()
		//		cleanup()
		os.Exit(1)
	})
	return nil
}

var devices []monome.Device

func main() {
	if os.Getenv("USER") != "root" {
		fmt.Fprintln(os.Stderr, "please run as root")
		os.Exit(1)
	}

	var err error

	devices, err = monome.Devices()

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
			err := do(d)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			}
		}(dev)
	}

	sigchan := make(chan os.Signal, 10)

	// listen for ctrl+c
	go signal.Notify(sigchan, os.Interrupt)

	// interrupt has happend
	<-sigchan

	fmt.Fprint(os.Stdout, "\ninterrupted, ")
	cleanup()
	os.Exit(0)
}

func cleanup() {
	fmt.Fprint(os.Stdout, "cleaning up...")

	for _, dev := range devices {
		dev.StopListening()
		//time.Sleep(time.Millisecond * 100)
		//dev.Close()
	}
	fmt.Fprintln(os.Stdout, "done")
}

// highlight the pressed buttons
func Handle(d monome.Device, x, y uint8, down bool) {
	var action = "released"
	if down {
		action = "pressed"
	}
	fmt.Printf("%s %s %v/%v\n", d, action, x, y)
	err := d.Switch(x, y, down)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}
}
