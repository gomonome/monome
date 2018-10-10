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
	connections      = map[string]monome.Connection{}
	sigchan          = make(chan os.Signal, 10)
	stopScanning     = make(chan bool)
	cleanup          = make(chan bool)
	removeConnection = make(chan monome.Connection, 4)
	addConnection    = make(chan monome.Connection, 4)

	cfg = config.MustNew("monome", "0.0.7", "demo a monome")

	rowCommand = cfg.MustCommand("row", "creates a row connection, based on all devices that could be found")
	argRowName = rowCommand.NewString("name", "name of the row device", config.Default("ROW"))

	scanCommand = cfg.MustCommand("scan", "scan for devices, allows to continously connect and disconnect to devices")
)

func initConnection(conn monome.Connection) error {
	var speed = time.Millisecond * 100
	if conn.String() == "monome64" {
		speed = time.Millisecond * 80
	}
	if conn.String() == "monome128" {
		speed = time.Millisecond * 50
	}
	err := monome.Marquee(conn, conn.String(), speed)
	if err != nil {
		return err
	}
	conn.SetHandler(monome.HandlerFunc(Handle))
	conn.StartListening(func(err error) {
		removeConnection <- conn
	})
	return nil
}

func manageConnections() {
	for {
		select {
		case conn := <-removeConnection:
			conn.StopListening()
			time.Sleep(time.Millisecond * 30)
			conn.Close()
			delete(connections, conn.String())
		case conn := <-addConnection:
			connections[conn.String()] = conn
			go func(d monome.Connection) {
				err := initConnection(d)
				if err != nil {
					fmt.Printf("ERROR: %v\n", err)
					removeConnection <- d
				}
			}(conn)
		case <-cleanup:
			fmt.Println("cleanup")
			for _, conn := range connections {
				conn.StopListening()
				time.Sleep(time.Millisecond * 100)
				conn.Close()
			}
			return
		default:
			runtime.Gosched()
		}
	}
}

func scanForConnections() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-stopScanning:
			fmt.Println("stop scanning")
			t.Stop()
			return
		case <-t.C:
			conns, _ := monome.Connections()

			for _, conn := range conns {
				addConnection <- conn
			}
		default:
			runtime.Gosched()

		}
	}
}

type Connections []monome.Connection

func (d Connections) Len() int {
	return len(d)
}

func (d Connections) Swap(a, b int) {
	d[a], d[b] = d[b], d[a]
}

func (d Connections) Less(a, b int) bool {
	return monome.NumButtons(d[a]) < monome.NumButtons(d[b])
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
		conns, _ := monome.Connections()

		if len(conns) == 0 {
			return fmt.Errorf("no monome devices found")
		}

		sconns := Connections(conns)
		sort.Sort(sconns)

		go manageConnections()
		addConnection <- monome.RowConnection(argRowName.Get(), []monome.Connection(sconns)...)

		// listen for ctrl+c
		go signal.Notify(sigchan, os.Interrupt)

		// interrupt has happend
		<-sigchan

		fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")
		cleanup <- true
		fmt.Fprint(os.Stdout, "done\n")
		os.Exit(0)

	case scanCommand:
		go manageConnections()
		go scanForConnections()

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
		conns, _ := monome.Connections()

		if len(conns) == 0 {
			return fmt.Errorf("no monome devices found")
		}
		go manageConnections()

		for _, conn := range conns {
			addConnection <- conn
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
func Handle(d monome.Connection, x, y uint8, down bool) {
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
