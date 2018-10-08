package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"time"

	"github.com/gomonome/monome"
	"github.com/metakeule/config"
)

var (
	devices      = map[string]monome.Device{}
	sigchan      = make(chan os.Signal, 10)
	stopScanning = make(chan bool)
	cleanup      = make(chan bool)
	removeDevice = make(chan monome.Device, 4)
	addDevice    = make(chan monome.Device, 4)

	cfg = config.MustNew("monome", "0.0.5", "demo a monome")

	rowCommand = cfg.MustCommand("row", "creates a rowdevice, based on all devices that could be found")
	argRowName = rowCommand.NewString("name", "name of the row device", config.Default("ROW"))

	scanCommand = cfg.MustCommand("scan", "scan for devices, allows to continously attach and detach devices")
)

func initDevice(dev monome.Device) error {
	var speed = time.Millisecond * 100
	if dev.String() == "monome64" {
		speed = time.Millisecond * 80
	}
	if dev.String() == "monome128" {
		speed = time.Millisecond * 50
	}
	err := dev.Marquee(dev.String(), speed)
	if err != nil {
		return err
	}
	dev.SetHandler(monome.HandlerFunc(Handle))
	dev.StartListening(func(err error) {
		removeDevice <- dev
	})
	return nil
}

func manageDevices() {
	for {
		select {
		case dev := <-removeDevice:
			dev.StopListening()
			time.Sleep(time.Millisecond * 30)
			dev.Close()
			delete(devices, dev.String())
		case dev := <-addDevice:
			devices[dev.String()] = dev
			go func(d monome.Device) {
				err := initDevice(d)
				if err != nil {
					fmt.Printf("ERROR: %v\n", err)
					removeDevice <- d
				}
			}(dev)
		case <-cleanup:
			fmt.Println("cleanup")
			for _, dev := range devices {
				dev.StopListening()
				time.Sleep(time.Millisecond * 100)
				dev.Close()
			}
			return
		default:
			runtime.Gosched()
		}
	}
}

func scanForDevices() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-stopScanning:
			fmt.Println("stop scanning")
			t.Stop()
			return
		case <-t.C:
			devs, _ := monome.Devices()

			for _, dev := range devs {
				addDevice <- dev
			}
		default:
			runtime.Gosched()

		}
	}
}

type Devices []monome.Device

func (d Devices) Len() int {
	return len(d)
}

func (d Devices) Swap(a, b int) {
	d[a], d[b] = d[b], d[a]
}

func (d Devices) Less(a, b int) bool {
	return d[a].NumButtons() < d[b].NumButtons()
}

func run() error {
	err := cfg.Run()

	if err != nil {
		return err
	}

	if os.Getenv("USER") != "root" {
		return fmt.Errorf("please run as root")
	}

	switch cfg.ActiveCommand() {
	case rowCommand:
		devs, _ := monome.Devices()

		if len(devs) == 0 {
			return fmt.Errorf("no monome devices found")
		}

		sdevs := Devices(devs)
		sort.Sort(sdevs)

		go manageDevices()
		addDevice <- monome.RowDevice(argRowName.Get(), []monome.Device(sdevs)...)

		// listen for ctrl+c
		go signal.Notify(sigchan, os.Interrupt)

		// interrupt has happend
		<-sigchan

		fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")
		cleanup <- true
		fmt.Fprint(os.Stdout, "done\n")
		os.Exit(0)

	case scanCommand:
		go manageDevices()
		go scanForDevices()

		// listen for ctrl+c
		go signal.Notify(sigchan, os.Interrupt)

		// interrupt has happend
		<-sigchan

		fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")
		stopScanning <- true
		cleanup <- true
		fmt.Fprint(os.Stdout, "done\n")
		os.Exit(0)
	default:
		devs, _ := monome.Devices()

		if len(devs) == 0 {
			return fmt.Errorf("no monome devices found")
		}
		go manageDevices()

		for _, dev := range devs {
			addDevice <- dev
		}

		// listen for ctrl+c
		go signal.Notify(sigchan, os.Interrupt)

		// interrupt has happend
		<-sigchan

		fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")
		cleanup <- true
		fmt.Fprint(os.Stdout, "done\n")
		os.Exit(0)
	}

	return nil

}

func main() {
	err := run()

	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

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
