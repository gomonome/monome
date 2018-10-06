package monome

import (
	"bytes"
	"sort"
	"time"
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

type multi struct {
	devices      []Device
	colToDev     sortByCol
	devToCol     map[int]uint8
	devNameToDev map[string]int
}

// Multi creates a large unified device out of a row of multiple devices.
// The order is from left to right.
// The number of columns is the sum of the columns of the devices.
// The number of rows is the smallest number of rows of any device.
func Multi(devices ...Device) Device {
	m := &multi{
		devices:      devices,
		devToCol:     map[int]uint8{},
		devNameToDev: map[string]int{},
	}
	m.calcOffsets()
	return m
}

func (m *multi) calcOffsets() {
	// find out the starting column for the device
	var startCol int
	for i, dev := range m.devices {
		m.colToDev = append(m.colToDev, [2]int{startCol, i})
		m.devToCol[i] = uint8(startCol)
		m.devNameToDev[dev.String()] = i
		startCol += int(dev.Cols())
	}
	sort.Sort(m.colToDev)
}

// Cols is the sum of the cols of the devices
func (m *multi) Cols() uint8 {
	var cols uint8

	for _, dev := range m.devices {
		cols += dev.Cols()
	}

	return cols
}

func (m *multi) On(x, y uint8)  { m.Set(x, y, 15) }
func (m *multi) Off(x, y uint8) { m.Set(x, y, 0) }

// Set sets the lights to the corresponding device
func (m *multi) Set(x, y, brightness uint8) {
	var dev int = 0
	var offset int = 0
	for _, mp := range m.colToDev {
		if mp[0] > int(y) {
			break
		}
		offset += mp[0]
		dev = mp[1]
	}
	m.devices[dev].Set(x, y-uint8(offset), brightness)
}

func (m *multi) SetHandler(h Handler) {
	for _, dev := range m.devices {
		dev.SetHandler(HandlerFunc(func(d Device, x, y uint8, down bool) {
			h.Handle(m, x, m.devToCol[m.devNameToDev[d.String()]]+y, down)
		}))
	}
}

func (m *multi) StartListening(errHandler func(error)) {
	for _, dev := range m.devices {
		dev.StartListening(errHandler)
	}
}

func (m *multi) StopListening() {
	for _, dev := range m.devices {
		dev.StopListening()
	}
}

func (m *multi) String() string {
	var bf bytes.Buffer
	bf.WriteString("<Multi ")
	for _, dev := range m.devices {
		bf.WriteString(dev.String() + "/")
	}
	bf.WriteString(">")
	return bf.String()
}

func (m *multi) Marquee(s string, dur time.Duration) {
	marquee(m, s, dur)
}

func (m *multi) Print(s string, dur time.Duration) {
	for _, dev := range m.devices {
		dev.Print(s, dur)
	}

}

func (m *multi) Read() error {
	panic("don't call me")
}

// NumButtons is the number of available buttons (cols*rows)
func (m *multi) NumButtons() uint8 {
	return m.Cols() * m.Rows()
}

// Rows returns the minimum of rows, each device has
func (m *multi) Rows() uint8 {
	var rows uint8
	for _, dev := range m.devices {
		if dev.Rows() < rows || rows == 0 {
			rows = dev.Rows()
		}
	}

	return rows
}

func (m *multi) AllOff() {
	for _, dev := range m.devices {
		dev.AllOff()
	}
}

func (m *multi) AllOn() {
	for _, dev := range m.devices {
		dev.AllOn()
	}
}

// Close closes all devices
func (m *multi) Close() error {
	for _, dev := range m.devices {
		dev.Close()
	}
	return nil
}

// IsClosed only returns true, if all devices are closed
func (m *multi) IsClosed() bool {
	for _, dev := range m.devices {
		if !dev.IsClosed() {
			return false
		}
	}
	return true
}
