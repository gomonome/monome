package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/gomonome/monome"
	"github.com/goosc/osc"
	"github.com/metakeule/config"
)

var (
	monomeConnection monome.Connection
	prefix           string
	listener         osc.Listener
	oscWriter        io.WriteCloser
	sigchan          = make(chan os.Signal, 10)
	stopScanning     = make(chan bool)
	cleanup          = make(chan bool, 4)
	newConnection    = make(chan monome.Connection, 4)

	cfg           = config.MustNew("monome", "0.0.7", "monome creates and OSC connection to the first available monome")
	argInaddress  = cfg.NewString("in", "address the monome is receiving from", config.Default("127.0.0.1:8082"))
	argOutaddress = cfg.NewString("out", "address the monome is sending to", config.Default("127.0.0.1:8002"))
	argPrefix     = cfg.NewString("prefix", "prefix for messages to address the monome device")
)

func main() {
	err := run()

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

}

func run() error {
	err := cfg.Run()

	if err != nil {
		return err
	}

	if os.Getenv("USER") != "root" {
		return fmt.Errorf("please run as root")
	}

	prefix = argPrefix.Get()

	listener, err = osc.UDPListener(argInaddress.Get())
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "listening on UDP %s\n", argInaddress.Get())

	oscWriter, err = osc.UDPWriter(argOutaddress.Get())

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "writing to UDP %s\n", argOutaddress.Get())

	go manageConnections()
	go scanForConnections()

	// listen for ctrl+c
	go signal.Notify(sigchan, os.Interrupt)

	// interrupt has happend
	<-sigchan

	fmt.Fprint(os.Stdout, "\ninterrupted...")
	//	cleanup <- true
	if monomeConnection != nil {
		fmt.Printf("closing %s", monomeConnection)
		listener.StopListening()
		monomeConnection.Close()
		oscWriter.Close()
	}
	fmt.Fprint(os.Stdout, "\ndone\n")
	os.Exit(0)

	return nil

}

type message struct {
	x          uint8
	y          uint8
	brightness uint8
}

type oscHandler struct{}

// for a full description of OSC messages for the monome, see
// https://monome.org/docs/osc/

/*
we only support
/grid/led/set x y s
/grid/led/level/set x y l
*/
func (o oscHandler) Matches(path osc.Path) bool {
	return true
}

func getMessage(values ...interface{}) message {
	var msg message

	for i, val := range values {
		if i > 2 {
			break
		}

		var what *uint8

		switch i {
		case 0:
			what = &msg.y
		case 1:
			what = &msg.x
		case 2:
			what = &msg.brightness
		}

		switch v := val.(type) {
		case int32:
			*what = uint8(v)
		case float32:
			val := uint8(v)
			if val == 0 && v > 0.0 {
				val = 1
			}
			*what = uint8(val)
		default:
			fmt.Fprintf(os.Stderr, "unsupported type: %T (%v)", v)

		}
	}
	return msg
}

func currentPrefix() string {
	return prefix
}

// Handle handles the incoming message
func (o oscHandler) Handle(path osc.Path, values ...interface{}) {
	pref := currentPrefix()
	switch path.String() {
	case pref + "/clear":
		if monomeConnection != nil {
			monome.SwitchAll(monomeConnection, false)
		}
	case pref + "/grid/led/intensity":
		// ignore
	case pref + "/tilt/set":
		// ignore
	case "/sys/prefix":
		prefix = values[1].(string)
	case pref + "/grid/led/set", prefix + "/led":
		var msg = getMessage(values...)
		if msg.brightness > 0 {
			msg.brightness = 15
		}
		sendMessage(msg)
	case pref + "/grid/led/level/set":
		var msg = getMessage(values...)
		sendMessage(msg)
	default:
		fmt.Fprintf(os.Stdout, "got unsupported OSC message: %q %v\n", path.String(), values)
	}

}

func initConnection(conn monome.Connection) error {
	monome.Greeter(conn)
	conn.SetHandler(monome.HandlerFunc(func(d monome.Connection, x, y uint8, down bool) {
		//   /grid/key x y s
		//   key state change at (x,y) to s (0 or 1, 1 = key down, 0 = key up).

		var downVal int32
		if down {
			downVal = 1
		}
		osc.III(prefix+"/grid/key").WriteTo(oscWriter, int32(y), int32(x), downVal)
	}))
	conn.StartListening(func(err error) {
		cleanup <- true
	})

	return listener.StartListening(oscHandler{})
}

func sendMessage(msg message) {
	monomeConnection.Set(msg.x, msg.y, msg.brightness)
}

func manageConnections() {
	for {
		select {
		case conn := <-newConnection:
			fmt.Fprintf(os.Stdout, "found: %s\n", conn.String())
			monomeConnection = conn
			stopScanning <- true
			err := initConnection(conn)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				monomeConnection.Close()
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
				newConnection <- conn
			}
		default:
			runtime.Gosched()

		}
	}
}
