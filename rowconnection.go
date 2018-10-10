package monome

import (
	"fmt"
	"sort"
)

type sortByCol [][2]int

func (c sortByCol) Len() int {
	return len(c)
}

func (c sortByCol) Less(a, b int) bool {
	return c[a][0] < c[b][0]
}

func (c sortByCol) Swap(a, b int) {
	c[a], c[b] = c[b], c[a]
}

var _ Connection = &rowConnection{}

type rowConnection struct {
	devices      []Connection
	colToDev     sortByCol
	devToCol     map[int]uint8
	devNameToDev map[string]int
	name         string
	cols         uint8
	rows         uint8
}

// RowConnection creates a unified connection out of a row of connections.
// The order is from left to right.
// The number of columns is the sum of the columns of the devices.
// The number of rows is the smallest number of rows of any device.
func RowConnection(name string, connections ...Connection) Connection {
	m := &rowConnection{
		devices:      connections,
		devToCol:     map[int]uint8{},
		devNameToDev: map[string]int{},
		name:         name,
	}
	if m.name == "" {
		m.name = "monome row"
	}
	m.calcOffsets()
	return m
}

func (m *rowConnection) calcOffsets() {
	// find out the starting column for the device
	var startCol int
	var cols uint8
	var rows uint8

	for i, dev := range m.devices {
		m.colToDev = append(m.colToDev, [2]int{startCol, i})
		m.devToCol[i] = uint8(startCol)
		m.devNameToDev[dev.String()] = i
		startCol += int(dev.Cols())
		cols += dev.Cols()
		if dev.Rows() < rows || rows == 0 {
			rows = dev.Rows()
		}
	}
	m.cols = cols
	m.rows = rows
	sort.Sort(m.colToDev)
}

// Rows returns the minimum of rows, each device has
func (m *rowConnection) Rows() uint8 {
	return m.rows
}

// Cols is the sum of the cols of the devices
func (m *rowConnection) Cols() uint8 {
	return m.cols
}

func (m *rowConnection) Switch(x, y uint8, on bool) error {
	var bightness uint8
	if on {
		bightness = 15
	}
	err := m.Set(x, y, bightness)
	if err == nil {
		return nil
	}
	e := err.(Error)
	if on {
		e.Task = fmt.Sprintf("switch on (%d/%d in row device)", x, y)
	} else {
		e.Task = fmt.Sprintf("switch off (%d/%d in row device)", x, y)
	}
	return e
}

// Set sets the lights to the corresponding device
func (m *rowConnection) Set(x, y, brightness uint8) error {
	var dev int = 0
	var offset int = 0
	for _, mp := range m.colToDev {
		if mp[0] > int(y) {
			break
		}
		offset += mp[0]
		dev = mp[1]
	}
	err := m.devices[dev].Set(x, y-uint8(offset), brightness)
	if err == nil {
		return nil
	}

	e := err.(Error)
	e.Task = fmt.Sprintf("set brightness to %d (%d/%d in row device)", brightness, x, y)
	return e
}

func (m *rowConnection) SetHandler(h Handler) {
	for _, dev := range m.devices {
		dev.SetHandler(HandlerFunc(func(d Connection, x, y uint8, down bool) {
			h.Handle(m, x, m.devToCol[m.devNameToDev[d.String()]]+y, down)
		}))
	}
}

func (m *rowConnection) StartListening(errHandler func(error)) {
	for _, dev := range m.devices {
		dev.StartListening(errHandler)
	}
}

func (m *rowConnection) StopListening() {
	for _, dev := range m.devices {
		dev.StopListening()
	}
}

func (m *rowConnection) String() string {
	return fmt.Sprintf("%s%d", m.name, NumButtons(m))
}

/*
func (m *rowDevice) Marquee(s string, dur time.Duration) error {
	return marquee(m, s, dur)
}
*/

/*
func (m *rowDevice) Print(s string, dur time.Duration) error {
	var errs Errors
	for _, dev := range m.devices {
		errs.Add(dev.Print(s, dur))
	}

	if errs.Len() == 0 {
		return nil
	}

	errs.Task = fmt.Sprintf("printing %q to row device %s", s, m.String())
	return &errs
}
*/

func (m *rowConnection) ReadMessage() error {
	panic("don't call me")
}

/*
// SwitchAll switches all lights on or off
// It returns the last error that happens and keeps trying to switch the rest as an error happens.
func (m *rowDevice) SwitchAll(on bool) error {
	var errs Errors
	for _, dev := range m.devices {
		errs.Add(SwitchAll(dev, on))
	}
	if errs.Len() == 0 {
		return nil
	}
	if on {
		errs.Task = "switch all on (row device)"
	} else {
		errs.Task = "switch all off (row device)"
	}
	return &errs
}
*/

// Close closes all devices
func (m *rowConnection) Close() error {
	var errs Errors
	for _, dev := range m.devices {
		errs.Add(dev.Close())
	}

	if errs.Len() == 0 {
		return nil
	}
	return &errs
}

// IsClosed only returns true, if all devices are closed
func (m *rowConnection) IsClosed() bool {
	for _, dev := range m.devices {
		if !dev.IsClosed() {
			return false
		}
	}
	return true
}
