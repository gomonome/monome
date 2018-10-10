package monome

import (
	"fmt"
	"io"
)

var _ Device = &testdevice{}

type Tester interface {
	Get() (x, y uint8, down bool, err error)
	Set(x, y, brightness uint8) error
	Name() string
	Cols() uint8
	Rows() uint8
	io.Closer
}

type getTester struct {
	rows uint8
	cols uint8
	get  func() (x, y uint8, down bool, err error)
}

func (g *getTester) Get() (x, y uint8, down bool, err error) {
	return g.get()
}

func (g *getTester) Set(x, y, brightness uint8) error {
	panic("do not call me")
}

func (g *getTester) Name() string {
	return "getTester"
}

func (g *getTester) Close() error {
	return nil
}

func (g *getTester) Cols() uint8 {
	return g.cols
}

func (g *getTester) Rows() uint8 {
	return g.rows
}

// GetTester returns a Tester tailored for testing the getting of the device
func GetTester(cols, rows uint8, get func() (x, y uint8, down bool, err error)) Tester {
	return &getTester{
		rows: rows,
		cols: cols,
		get:  get,
	}
}

type setTester struct {
	rows uint8
	cols uint8
	set  func(x, y, brightness uint8) error
}

func (g *setTester) Get() (x, y uint8, down bool, err error) {
	panic("do not call me")
}

func (g *setTester) Set(x, y, brightness uint8) error {
	return g.set(x, y, brightness)
}

func (g *setTester) Name() string {
	return "setTester"
}

func (g *setTester) Close() error {
	return nil
}

func (g *setTester) Cols() uint8 {
	return g.cols
}

func (g *setTester) Rows() uint8 {
	return g.rows
}

// SetTester returns a Tester tailored for testing the setting of the device
func SetTester(cols, rows uint8, set func(x, y, brightness uint8) error) Tester {
	return &setTester{
		rows: rows,
		cols: cols,
		set:  set,
	}
}

type closeTester struct {
	rows uint8
	cols uint8
	clos func() error
}

func (g *closeTester) Get() (x, y uint8, down bool, err error) {
	panic("do not call me")
}

func (g *closeTester) Set(x, y, brightness uint8) error {
	panic("do not call me")
}

func (g *closeTester) Name() string {
	return "closeTester"
}

func (g *closeTester) Cols() uint8 {
	return g.cols
}

func (g *closeTester) Rows() uint8 {
	return g.rows
}

func (g *closeTester) Close() error {
	return g.clos()
}

// CloseTester returns a Tester tailored for testing the closing of the device
func CloseTester(cols, rows uint8, clos func() error) Tester {
	return &closeTester{
		rows: rows,
		cols: cols,
		clos: clos,
	}
}

type testdevice struct {
	mn     *connection
	tester Tester
}

func (t *testdevice) String() string {
	return t.tester.Name()
}

func (m *testdevice) Rows() uint8 { return m.tester.Rows() }
func (m *testdevice) Cols() uint8 { return m.tester.Cols() }
func (m *testdevice) Switch(x, y uint8, on bool) error {
	var brightness uint8
	if on {
		brightness = 15
	}
	err := m.Set(x, y, brightness)
	if err == nil {
		return nil
	}

	e := err.(Error)
	if on {
		e.Task = "switch on"
	} else {
		e.Task = "switch off"
	}
	return e
}

func (m *testdevice) Set(x, y, brightness uint8) error {
	err := m.tester.Set(x, y, brightness)
	if err != nil {
		var e Error
		e.Device = m.String()
		e.X = x
		e.Y = y
		e.WrappedError = err
		e.Task = fmt.Sprintf("set brightness to %d", brightness)
		return e
	}
	return nil
}

func (m *testdevice) ReadMessage() error {
	x, y, down, err := m.tester.Get()
	if err != nil {
		return err
	}
	m.mn.h.Handle(m.mn, x, y, down)
	return nil
}

// TestDevice returns a new (fake) monome device, based on the given tester
func TestDevice(tester Tester, options ...Option) Connection {
	var m = &connection{
		dev:          tester,
		pollInterval: defaultPollInterval,
	}

	for _, opt := range options {
		opt(m)
	}

	m.Device = &testdevice{m, tester}
	return m
}
