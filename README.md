# monome
Go library to program the monome

[![Documentation](http://godoc.org/github.com/gomonome/monome?status.png)](http://godoc.org/github.com/gomonome/monome)

## Installation

It is recommended to use Go 1.11 with module support (`$GO111MODULE=on`).

```
go get -d github.com/gomonome/monome/...
```

## Example

```go
package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/gomonome/monome"
)

func setup(conn monome.Connection) {
	monome.Greeter(conn)
	conn.SetHandler(monome.HandlerFunc(func(c monome.Connection, x, y uint8, down bool) {
		action := "released"
		if down {
			action = "pressed"
		}
		fmt.Printf("%s %s key %v/%v\n", c, action, x, y)

		// switch the lights
		c.Switch(x, y, down)
	}))
	conn.StartListening(func(err error) {
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

	conns, _ := monome.Connections()

	if len(conns) < 1 {
		fmt.Fprintln(os.Stderr, "no monome device found")
		os.Exit(1)
	}

	for _, conn := range conns {
		setup(conn)
	}

	// listen for ctrl+c
	go signal.Notify(sigchan, os.Interrupt)

	// interrupt has happend
	<-sigchan

	fmt.Fprint(os.Stdout, "\ninterrupted, cleaning up...")

	for _, conn := range conns {
		conn.Close()
	}

	fmt.Fprintln(os.Stdout, "done")
	os.Exit(0)
}

```


## License

MIT (see LICENSE file) 
